package service

import (
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
	"mycocycle-growbag-transfer-gate/internal/maturity"
)

// DeviceReading submits a single device invocation. Scripted device failures
// only produce a retryable device attempt; a successful device result is parsed
// into a fixed-point reading that is only accepted when length, sign, scale and
// range all validate.
func (s *Service) DeviceReading(id domain.TaskID, req DeviceReadingRequest) (DeviceReadingResult, error) {
	t, err := s.loadTask(id)
	if err != nil {
		return DeviceReadingResult{}, err
	}
	if err := s.guard(t, req.Generation); err != nil {
		return DeviceReadingResult{}, err
	}
	if t.State != inspection.StateObserving && !isVerifying(t.State) {
		return DeviceReadingResult{}, domain.NewError(domain.CodeFinalStateRejected,
			"task not ready for readings", string(t.State))
	}

	outcome := s.scripts.Result(req.DeviceType, req.DeviceID, req.ScriptSeq)
	prior := s.countPriorAttempts(id, req.DeviceType, req.DeviceID)
	retryCount := prior + 1

	attempt := &domain.DeviceAttempt{
		DeviceType: req.DeviceType,
		DeviceID:   req.DeviceID,
		Object:     string(req.Metric),
		TaskID:     id,
		Generation: req.Generation,
		At:         s.tick(),
		ScriptSeq:  req.ScriptSeq,
		Result:     outcome,
		RetryCount: retryCount,
	}
	if outcome != domain.AttemptAccepted {
		attempt.ErrorCode = codeForAttempt(outcome)
		if err := s.store.SaveAttempt(ctx(), attempt); err != nil {
			return DeviceReadingResult{}, err
		}
		s.audit(id, req.Generation, "", domain.AuditDeviceAttempt, string(outcome))
		return DeviceReadingResult{Result: outcome, RetryCount: retryCount, State: t.State}, nil
	}

	// Device accepted: parse the value into a fixed-point reading.
	reading, status := s.parseReading(id, req)
	if reading != nil && status == maturity.ReadingAccepted {
		reading.Derived = deriveJudgment(t.Snapshot, reading)
	}
	if err := s.store.SaveAttempt(ctx(), attempt); err != nil {
		return DeviceReadingResult{}, err
	}
	if reading != nil {
		if err := s.store.SaveReading(ctx(), reading); err != nil {
			return DeviceReadingResult{}, err
		}
		s.audit(id, req.Generation, "", domain.AuditReadingRecorded, string(req.Metric))
	}

	state := t.State
	if status == maturity.ReadingAccepted && state == inspection.StateVerifyingPhysChem {
		readings, err := s.store.LoadReadings(ctx(), id)
		if err != nil {
			return DeviceReadingResult{}, err
		}
		if physChemCollected(t.Snapshot.BagPositions, readings) {
			if err := s.advanceState(id, t, inspection.StatePendingReview); err != nil {
				return DeviceReadingResult{}, err
			}
			state = inspection.StatePendingReview
		}
	}

	return DeviceReadingResult{Result: outcome, RetryCount: retryCount, State: state}, nil
}

func (s *Service) countPriorAttempts(id domain.TaskID, dt domain.DeviceType, dev domain.DeviceID) int {
	attempts, err := s.store.LoadAttempts(ctx(), id)
	if err != nil {
		return 0
	}
	n := 0
	for _, a := range attempts {
		if a.DeviceType == dt && a.DeviceID == dev {
			n++
		}
	}
	return n
}

func codeForAttempt(res domain.AttemptResult) domain.ErrorCode {
	switch res {
	case domain.AttemptRejected:
		return domain.CodeDeviceFailure
	case domain.AttemptDisconnected:
		return domain.CodeDeviceFailure
	case domain.AttemptTimeout:
		return domain.CodeDeviceFailure
	case domain.AttemptFormatError:
		return domain.CodeReadingOutOfRange
	default:
		return domain.CodeDeviceFailure
	}
}

// parseReading parses an accepted device value into a reading, returning the
// reading (nil on parse failure) and its status.
func (s *Service) parseReading(id domain.TaskID, req DeviceReadingRequest) (*maturity.PhysChemReading, maturity.ReadingStatus) {
	scale, ok := domain.MetricScale(req.Metric)
	if !ok {
		return nil, maturity.ReadingParseError
	}
	f, err := domain.ParseFixedNonNegative(req.Value, scale)
	status := maturity.ReadingAccepted
	switch err {
	case nil:
	case domain.ErrOverflow:
		status = maturity.ReadingOverflow
	case domain.ErrValueOutOfRange:
		status = maturity.ReadingOutOfRange
	default:
		status = maturity.ReadingParseError
	}
	r := &maturity.PhysChemReading{
		TaskID:       id,
		Generation:   req.Generation,
		Position:     req.Position,
		DayAge:       req.DayAge,
		Metric:       req.Metric,
		Value:        f,
		SourceDevice: req.DeviceID,
		Status:       status,
		Derived:      maturity.DerivedNone,
	}
	if status == maturity.ReadingAccepted {
		if verr := domain.ValidateMetricValue(req.Metric, f); verr != nil {
			status = maturity.ReadingOutOfRange
			r.Status = status
		}
	}
	return r, status
}

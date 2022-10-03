// Copyright 2022 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Package rangelog implements kvserver.RangeLogWriter
package rangelog

import (
	"context"
	"encoding/json"

	"github.com/cockroachdb/cockroach/pkg/kv"
	"github.com/cockroachdb/cockroach/pkg/kv/kvserver/kvserverpb"
	"github.com/cockroachdb/cockroach/pkg/security/username"
	"github.com/cockroachdb/cockroach/pkg/sql/sessiondata"
	"github.com/cockroachdb/cockroach/pkg/sql/sqlutil"
	"github.com/cockroachdb/errors"
)

// Writer implements kvserver.RangeLogWriter using the InternalExecutor.
type Writer struct {
	ie sqlutil.InternalExecutor
}

// NewWriter returns a new Writer which implements kvserver.RangeLogWriter
// using the InternalExecutor.
func NewWriter(ie sqlutil.InternalExecutor) *Writer {
	return &Writer{ie: ie}
}

// WriteRangeLogEvent implements kvserver.RangeLogWriter. It writes the event
// to the system.rangelog table in the provided transaction.
func (s *Writer) WriteRangeLogEvent(
	ctx context.Context, txn *kv.Txn, event kvserverpb.RangeLogEvent,
) error {
	const insertEventTableStmt = `
	INSERT INTO system.rangelog (
		timestamp, "rangeID", "storeID", "eventType", "otherRangeID", info
	)
	VALUES(
		$1, $2, $3, $4, $5, $6
	)
	`
	args := []interface{}{
		event.Timestamp,
		event.RangeID,
		event.StoreID,
		event.EventType.String(),
		nil, // otherRangeID
		nil, // info
	}
	if event.OtherRangeID != 0 {
		args[4] = event.OtherRangeID
	}
	if event.Info != nil {
		infoBytes, err := json.Marshal(*event.Info)
		if err != nil {
			return err
		}
		args[5] = string(infoBytes)
	}

	rows, err := s.ie.ExecEx(ctx, "log-range-event", txn,
		sessiondata.InternalExecutorOverride{User: username.RootUserName()},
		insertEventTableStmt, args...)
	if err != nil {
		return err
	}
	if rows != 1 {
		return errors.Errorf("%d rows affected by log insertion; expected exactly one row affected.", rows)
	}
	return nil
}

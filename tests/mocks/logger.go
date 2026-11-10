package mocks

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

type MockLogger struct {
	ctrl     *gomock.Controller
	recorder *MockLoggerMockRecorder
	isgomock struct{}
}

type MockLoggerMockRecorder struct {
	mock *MockLogger
}

func NewMockLogger(ctrl *gomock.Controller) *MockLogger {
	mock := &MockLogger{ctrl: ctrl}
	mock.recorder = &MockLoggerMockRecorder{mock}
	return mock
}

func (m *MockLogger) EXPECT() *MockLoggerMockRecorder {
	return m.recorder
}

func (m *MockLogger) Debug(msg string, fields map[string]any) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Debug", msg, fields)
}

func (mr *MockLoggerMockRecorder) Debug(msg, fields any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Debug", reflect.TypeOf((*MockLogger)(nil).Debug), msg, fields)
}

func (m *MockLogger) DebugContext(ctx context.Context, msg string, fields map[string]any) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "DebugContext", ctx, msg, fields)
}

func (mr *MockLoggerMockRecorder) DebugContext(ctx, msg, fields any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DebugContext", reflect.TypeOf((*MockLogger)(nil).DebugContext), ctx, msg, fields)
}

func (m *MockLogger) Error(msg string, fields map[string]any) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Error", msg, fields)
}

func (mr *MockLoggerMockRecorder) Error(msg, fields any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Error", reflect.TypeOf((*MockLogger)(nil).Error), msg, fields)
}

func (m *MockLogger) ErrorContext(ctx context.Context, msg string, fields map[string]any) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ErrorContext", ctx, msg, fields)
}

func (mr *MockLoggerMockRecorder) ErrorContext(ctx, msg, fields any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ErrorContext", reflect.TypeOf((*MockLogger)(nil).ErrorContext), ctx, msg, fields)
}

func (m *MockLogger) Info(msg string, fields map[string]any) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Info", msg, fields)
}

func (mr *MockLoggerMockRecorder) Info(msg, fields any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Info", reflect.TypeOf((*MockLogger)(nil).Info), msg, fields)
}

func (m *MockLogger) InfoContext(ctx context.Context, msg string, fields map[string]any) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "InfoContext", ctx, msg, fields)
}

func (mr *MockLoggerMockRecorder) InfoContext(ctx, msg, fields any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InfoContext", reflect.TypeOf((*MockLogger)(nil).InfoContext), ctx, msg, fields)
}

func (m *MockLogger) Warn(msg string, fields map[string]any) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Warn", msg, fields)
}

func (mr *MockLoggerMockRecorder) Warn(msg, fields any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warn", reflect.TypeOf((*MockLogger)(nil).Warn), msg, fields)
}

func (m *MockLogger) WarnContext(ctx context.Context, msg string, fields map[string]any) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "WarnContext", ctx, msg, fields)
}

func (mr *MockLoggerMockRecorder) WarnContext(ctx, msg, fields any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WarnContext", reflect.TypeOf((*MockLogger)(nil).WarnContext), ctx, msg, fields)
}

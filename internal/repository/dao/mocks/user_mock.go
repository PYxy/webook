// Code generated by MockGen. DO NOT EDIT.
// Source: F:\git_push\webook\internal\repository\dao\type.go

// Package daomocks is a generated GoMock package.
package daomocks

import (
	context "context"
	reflect "reflect"

	dao "gitee.com/geekbang/basic-go/webook/internal/repository/dao"
	gomock "go.uber.org/mock/gomock"
)

// MockUserDaoInterface is a mock of UserDaoInterface interface.
type MockUserDaoInterface struct {
	ctrl     *gomock.Controller
	recorder *MockUserDaoInterfaceMockRecorder
}

// MockUserDaoInterfaceMockRecorder is the mock recorder for MockUserDaoInterface.
type MockUserDaoInterfaceMockRecorder struct {
	mock *MockUserDaoInterface
}

// NewMockUserDaoInterface creates a new mock instance.
func NewMockUserDaoInterface(ctrl *gomock.Controller) *MockUserDaoInterface {
	mock := &MockUserDaoInterface{ctrl: ctrl}
	mock.recorder = &MockUserDaoInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserDaoInterface) EXPECT() *MockUserDaoInterfaceMockRecorder {
	return m.recorder
}

// FindByEmail mocks base method.
func (m *MockUserDaoInterface) FindByEmail(ctx context.Context, email string) (dao.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByEmail", ctx, email)
	ret0, _ := ret[0].(dao.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByEmail indicates an expected call of FindByEmail.
func (mr *MockUserDaoInterfaceMockRecorder) FindByEmail(ctx, email interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByEmail", reflect.TypeOf((*MockUserDaoInterface)(nil).FindByEmail), ctx, email)
}

// FindById mocks base method.
func (m *MockUserDaoInterface) FindById(ctx context.Context, id int64) (dao.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindById", ctx, id)
	ret0, _ := ret[0].(dao.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindById indicates an expected call of FindById.
func (mr *MockUserDaoInterfaceMockRecorder) FindById(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindById", reflect.TypeOf((*MockUserDaoInterface)(nil).FindById), ctx, id)
}

// FindByPhone mocks base method.
func (m *MockUserDaoInterface) FindByPhone(ctx context.Context, phone string) (dao.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByPhone", ctx, phone)
	ret0, _ := ret[0].(dao.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByPhone indicates an expected call of FindByPhone.
func (mr *MockUserDaoInterfaceMockRecorder) FindByPhone(ctx, phone interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByPhone", reflect.TypeOf((*MockUserDaoInterface)(nil).FindByPhone), ctx, phone)
}

// FindByWeChat mocks base method.
func (m *MockUserDaoInterface) FindByWeChat(ctx context.Context, openId string) (dao.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByWeChat", ctx, openId)
	ret0, _ := ret[0].(dao.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByWeChat indicates an expected call of FindByWeChat.
func (mr *MockUserDaoInterfaceMockRecorder) FindByWeChat(ctx, openId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByWeChat", reflect.TypeOf((*MockUserDaoInterface)(nil).FindByWeChat), ctx, openId)
}

// Insert mocks base method.
func (m *MockUserDaoInterface) Insert(ctx context.Context, u dao.User) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Insert", ctx, u)
	ret0, _ := ret[0].(error)
	return ret0
}

// Insert indicates an expected call of Insert.
func (mr *MockUserDaoInterfaceMockRecorder) Insert(ctx, u interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Insert", reflect.TypeOf((*MockUserDaoInterface)(nil).Insert), ctx, u)
}

// Update mocks base method.
func (m *MockUserDaoInterface) Update(ctx context.Context, u dao.User) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, u)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockUserDaoInterfaceMockRecorder) Update(ctx, u interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockUserDaoInterface)(nil).Update), ctx, u)
}

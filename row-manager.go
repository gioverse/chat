package chat

import "gioui.org/layout"

// RowID uniquely identifies a row of content.
type RowID string

// NoID is a special ID that can be used by Rows that do not require
// a unique identifier. Only stateless rows may go without a unique
// identifier.
const NoID = RowID("")

// Row is a type that can be presented by a RowManager.
type Row interface {
	// ID returns a unique identifier for the Row, if it has one.
	// In order for a Row to be stateful, it _must_ return a unique
	// ID. Rows that are not stateful may return the special ID
	// NoID to indicate that they do not need any state allocated
	// for them.
	ID() RowID
}

// Presenter is a function that can transform the data for a Row
// into a widget to be laid out in the user interface.
type Presenter func(current Row, state interface{}) layout.Widget

// Allocator is a function that can allocate the appropriate state
// type for a given Row.
type Allocator func(current Row) (state interface{})

// RowManager presents heterogenous Row data. Each row could represent
// any element of an interface that occupies a horizontal slice of
// screen real-estate.
type RowManager struct {
	// Rows is the list of data to present.
	Rows []Row
	// presenter is a function that can transform a single Row into
	// a presentable widget.
	presenter Presenter
	// allocator is a function that can instantiate the state for a particular
	// Row.
	allocator Allocator
	// rowState is a map storing the state for the Rows managed
	// by the manager.
	rowState map[RowID]interface{}
}

// NewManager constructs a manager with the given allocator and presenter.
func NewManager(allocator Allocator, presenter Presenter) *RowManager {
	return &RowManager{
		presenter: presenter,
		allocator: allocator,
		rowState:  make(map[RowID]interface{}),
	}
}

// Layout the Row at position index within the manager's Row list.
func (m *RowManager) Layout(gtx layout.Context, index int) layout.Dimensions {
	data := m.Rows[index]
	id := data.ID()
	state, ok := m.rowState[id]
	if !ok && id != NoID {
		state = m.allocator(data)
		m.rowState[id] = state
	}
	widget := m.presenter(data, state)
	return widget(gtx)
}

// Len returns the number of rows managed by this manager.
func (m *RowManager) Len() int {
	return len(m.Rows)
}

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlwaysYes(t *testing.T) {
	confirm := AlwaysYes()

	result, err := confirm("anything")

	require.NoError(t, err)
	assert.True(t, result)
}

func TestMockConfirm(t *testing.T) {
	confirm := mockConfirm(true)
	result, err := confirm("prompt")
	require.NoError(t, err)
	assert.True(t, result)

	confirm = mockConfirm(false)
	result, err = confirm("prompt")
	require.NoError(t, err)
	assert.False(t, result)
}

func TestMockConfirmSequence(t *testing.T) {
	confirm := mockConfirmSequence(true, false, true)

	r1, err := confirm("first")
	require.NoError(t, err)
	assert.True(t, r1)

	r2, err := confirm("second")
	require.NoError(t, err)
	assert.False(t, r2)

	r3, err := confirm("third")
	require.NoError(t, err)
	assert.True(t, r3)
}

func TestMockSelect(t *testing.T) {
	sel := mockSelect(2)
	idx, err := sel("pick one", []string{"a", "b", "c"})
	require.NoError(t, err)
	assert.Equal(t, 2, idx)
}

func TestMockSelectSequence(t *testing.T) {
	sel := mockSelectSequence(0, 2, 1)

	idx, err := sel("first", []string{"a", "b", "c"})
	require.NoError(t, err)
	assert.Equal(t, 0, idx)

	idx, err = sel("second", []string{"a", "b", "c"})
	require.NoError(t, err)
	assert.Equal(t, 2, idx)

	idx, err = sel("third", []string{"a", "b", "c"})
	require.NoError(t, err)
	assert.Equal(t, 1, idx)
}

func TestMockMultiSelect(t *testing.T) {
	ms := mockMultiSelect([]int{0, 2, 4})
	indices, err := ms("pick many", []string{"a", "b", "c", "d", "e"})
	require.NoError(t, err)
	assert.Equal(t, []int{0, 2, 4}, indices)
}

func TestMockPrompt(t *testing.T) {
	p := mockPrompt("hello", "world")

	r1, err := p("first")
	require.NoError(t, err)
	assert.Equal(t, "hello", r1)

	r2, err := p("second")
	require.NoError(t, err)
	assert.Equal(t, "world", r2)

	_, err = p("third")
	assert.Error(t, err, "should error when no more responses")
}

func TestNewPromptKit(t *testing.T) {
	kit := NewPromptKit()
	assert.NotNil(t, kit.Prompt)
	assert.NotNil(t, kit.Confirm)
	assert.NotNil(t, kit.Select)
	assert.NotNil(t, kit.MultiSelect)
}

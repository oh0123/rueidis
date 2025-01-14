// Code generated DO NOT EDIT

package cmds

import "testing"

func sentinel0(s Builder) {
	s.SentinelFailover().Master("1").Build()
	s.SentinelGetMasterAddrByName().Master("1").Build()
	s.SentinelReplicas().Master("1").Build()
	s.SentinelSentinels().Master("1").Build()
}

func TestCommand_InitSlot_sentinel(t *testing.T) {
	var s = NewBuilder(InitSlot)
	t.Run("0", func(t *testing.T) { sentinel0(s) })
}

func TestCommand_NoSlot_sentinel(t *testing.T) {
	var s = NewBuilder(NoSlot)
	t.Run("0", func(t *testing.T) { sentinel0(s) })
}

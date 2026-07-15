package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecialRemovalCanHideUserIdentityGroup(t *testing.T) {
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	specialGroups := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup
	originalSpecialGroups := specialGroups.ReadAll()
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
		specialGroups.Clear()
		specialGroups.AddAll(originalSpecialGroups)
	})

	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"test-route":"Test route"}`))
	specialGroups.Clear()
	specialGroups.Set("test-member", map[string]string{
		"-:test-member": "",
	})

	groups := GetUserUsableGroups("test-member")

	assert.NotContains(t, groups, "test-member")
	assert.Equal(t, "Test route", groups["test-route"])
}

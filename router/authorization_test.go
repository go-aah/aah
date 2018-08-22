// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/router source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"errors"
	"testing"

	"aahframe.work/aah/config"
	"aahframe.work/aah/security"
	"aahframe.work/aah/security/authz"
	"github.com/stretchr/testify/assert"
)

func TestRouteAuthorizationConfig(t *testing.T) {
	cfg, err := config.ParseString(`
    user_info {
      authorization {
        roles = [
          "hasrole(manager)",
          "hasanyrole(role1, role2, role3)"
        ]

        permissions = [
          "ispermitted(newsletter:read,write)",
          "ispermittedall(newsletter:read,write | newsletter:12345)"
        ]
      }
    }
  `)
	assert.Nil(t, err)

	info, err := parseAuthorizationInfo(cfg, "user_info", &parentRouteInfo{AuthorizationInfo: &authorizationInfo{Satisfy: "either"}})
	assert.Nil(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "either", info.Satisfy)
	assert.Equal(t, []string{"manager"}, info.Roles["hasrole"])
	assert.Equal(t, []string{"role1", "role2", "role3"}, info.Roles["hasanyrole"])
	assert.Equal(t, []string{"newsletter:read,write"}, info.Permissions["ispermitted"])
	assert.Equal(t, []string{"newsletter:read,write", "newsletter:12345"}, info.Permissions["ispermittedall"])
	assert.True(t, len(info.String()) > 0)
}

func TestRouteAuthorizationConfigParentRoute(t *testing.T) {
	cfg, err := config.ParseString(`
    user_info {
    }
  `)
	assert.Nil(t, err)

	info, err := parseAuthorizationInfo(cfg, "user_info", &parentRouteInfo{
		AuthorizationInfo: &authorizationInfo{
			Satisfy: "either",
			Roles: map[string][]string{
				"hasrole":    []string{"manager"},
				"hasanyrole": []string{"role1", "role2", "role3"},
			},
			Permissions: map[string][]string{
				"ispermitted":    []string{"newsletter:read,write"},
				"ispermittedall": []string{"newsletter:read,write", "newsletter:12345"},
			},
		},
	})
	assert.Nil(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "either", info.Satisfy)
	assert.Equal(t, []string{"manager"}, info.Roles["hasrole"])
	assert.Equal(t, []string{"role1", "role2", "role3"}, info.Roles["hasanyrole"])
	assert.Equal(t, []string{"newsletter:read,write"}, info.Permissions["ispermitted"])
	assert.Equal(t, []string{"newsletter:read,write", "newsletter:12345"}, info.Permissions["ispermittedall"])
}

func TestRouteAuthorizationConfigErrorRolesPermissions(t *testing.T) {
	testcases := []struct {
		label     string
		configStr string
		err       error
	}{
		{
			label: "Test missing right bracket",
			configStr: `
        user_info {
          authorization {
            roles = [
              "hasrole(manager",
              "hasanyrole(role1, role2, role3)"
            ]
          }
        }
      `,
			err: errors.New("user_info.authorization.roles at index 1 have incorrect open/close brackets 'hasrole(manager'"),
		},
		{
			label: "Test additional left bracket",
			configStr: `
        user_info {
          authorization {
            roles = [
              "hasrole((manager)",
              "hasanyrole(role1, role2, role3)"
            ]
          }
        }
      `,
			err: errors.New("user_info.authorization.roles at index 1 have incorrect open/close brackets 'hasrole((manager)'"),
		},
		{
			label: "Test missing both brackets and inputs",
			configStr: `
        user_info {
          authorization {
            roles = [
              "hasrole(manager)",
              "hasanyrole"
            ]
          }
        }
      `,
			err: errors.New("user_info.authorization.roles at index 2 have incorrect open/close brackets 'hasanyrole'"),
		},
		{
			label: "Test missing inputs",
			configStr: `
        user_info {
          authorization {
            roles = [
              "hasrole(manager)",
              "hasanyrole()"
            ]
          }
        }
      `,
			err: errors.New("user_info.authorization.roles at index 2 have func 'hasanyrole()' without input"),
		},
		{
			label: "Test hasrole has more than one input",
			configStr: `
        user_info {
          authorization {
            roles = [
              "hasrole(manager, role1)"
            ]
          }
        }
      `,
			err: errors.New("user_info.authorization.roles at index 1 have func 'hasrole' supports only one input param"),
		},
		{
			label: "Test satisfy is both and permission is not configured",
			configStr: `
        user_info {
          authorization {
            satisfy = "both"
            roles = [
              "hasrole(manager)"
            ]
          }
        }
      `,
			err: errors.New("user_info.authorization.satisfy configured as 'both', however roles and permissions is not configured"),
		},
		{
			label: "Test ispermitted has more than one input",
			configStr: `
        user_info {
          authorization {
            permissions = [
              "ispermitted(newsletter:read,write | newsletter:12345)"
            ]
          }
        }
      `,
			err: errors.New("user_info.authorization.permissions at index 1 have func 'ispermitted' supports only one input param"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.label, func(t *testing.T) {
			cfg, err := config.ParseString(tc.configStr)
			assert.Nil(t, err)

			info, err := parseAuthorizationInfo(cfg, "user_info", &parentRouteInfo{AuthorizationInfo: &authorizationInfo{Satisfy: "either"}})
			assert.NotNil(t, err)
			assert.Nil(t, info)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestHasAccess(t *testing.T) {
	allfuncsCfgStr := `
    user_info {
      authorization {
        roles = [
          "hasrole(manager)",
          "hasanyrole(role1, role2, role3)",
          "hasallroles(manager, role3)"
        ]

        permissions = [
          "ispermitted(newsletter:read,write)",
          "ispermittedall(newsletter:read,write | newsletter:12345)"
        ]
      }
    }
  `

	testcases := []struct {
		label              string
		configStr          string
		satisfy            string
		subjectRoles       []string
		subjectPermissions []string
		result             bool
	}{
		{
			label:              "All roles func and permissions func",
			configStr:          allfuncsCfgStr,
			satisfy:            "either",
			subjectRoles:       []string{"manager", "role3"},
			subjectPermissions: []string{"newsletter:read,write", "newsletter:12345"},
			result:             true,
		},
		{
			label:        "Only roles check, satisfy: either",
			configStr:    allfuncsCfgStr,
			satisfy:      "either",
			subjectRoles: []string{"manager", "role3"},
			result:       true,
		},
		{
			label:        "Only roles check, satisfy: both",
			configStr:    allfuncsCfgStr,
			satisfy:      "both",
			subjectRoles: []string{"manager", "role3"},
		},
		{
			label:              "Only permissions check, satisfy: either",
			configStr:          allfuncsCfgStr,
			satisfy:            "either",
			subjectPermissions: []string{"newsletter:read,write", "newsletter:12345"},
			result:             true,
		},
		{
			label:              "Only permissions check, satisfy: both",
			configStr:          allfuncsCfgStr,
			satisfy:            "both",
			subjectPermissions: []string{"newsletter:read,write", "newsletter:12345"},
		},
		{
			label:              "All roles and permissions not exists",
			configStr:          allfuncsCfgStr,
			satisfy:            "either",
			subjectRoles:       []string{"notexist1", "notexist2"},
			subjectPermissions: []string{"notexist:read,write", "notexist:12345"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.label, func(t *testing.T) {
			cfg, err := config.ParseString(tc.configStr)
			assert.Nil(t, err)

			authInfo, err := parseAuthorizationInfo(cfg, "user_info", &parentRouteInfo{AuthorizationInfo: &authorizationInfo{Satisfy: tc.satisfy}})
			assert.Nil(t, err)
			assert.NotNil(t, authInfo)

			r := &Route{authorizationInfo: authInfo}
			result, _ := r.HasAccess(createSubject(tc.subjectRoles, tc.subjectPermissions))
			assert.Equal(t, tc.result, result)
		})
	}

	r := &Route{}
	result, _ := r.HasAccess(createSubject([]string{"manager", "role3"}, []string{}))
	assert.True(t, result)
}

func createSubject(roles []string, permissions []string) *security.Subject {
	authInfo := authz.NewAuthorizationInfo()

	authInfo.AddRole(roles...)
	authInfo.AddPermissionString(permissions...)

	return &security.Subject{
		AuthorizationInfo: authInfo,
	}
}

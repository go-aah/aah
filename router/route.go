// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/router source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"bytes"
	"fmt"
	"strings"

	"aahframe.work/aah/config"
	"aahframe.work/aah/security"
	"aahframe.work/aah/security/authz"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Route
//______________________________________________________________________________

// Route holds the single route details.
type Route struct {
	IsAntiCSRFCheck bool
	IsStatic        bool
	ListDir         bool
	MaxBodySize     int64
	Name            string
	Path            string
	Method          string
	Target          string
	Action          string
	ParentName      string
	Auth            string
	Dir             string
	File            string
	CORS            *CORS
	Constraints     map[string]string

	authorizationInfo *authorizationInfo
}

// IsDir method returns true if serving directory otherwise false.
func (r *Route) IsDir() bool {
	return len(r.Dir) > 0 && len(r.File) == 0
}

// IsFile method returns true if serving single file otherwise false.
func (r *Route) IsFile() bool {
	return len(r.File) > 0
}

// HasAccess method does authorization check based on configured values at route
// level.
// TODO: the appropriate place for this method would be `security` package.
func (r *Route) HasAccess(subject *security.Subject) (bool, []*authz.Reason) {
	var reasons []*authz.Reason
	if r.authorizationInfo == nil || (len(r.authorizationInfo.Roles) == 0 &&
		len(r.authorizationInfo.Permissions) == 0) {
		// Possibly aah User might be doing authroization at controller manually
		return true, reasons
	}

	// Check roles
	rolesResult := true
	for fn, inputs := range r.authorizationInfo.Roles {
		switch fn {
		case "hasrole":
			rolesResult = subject.HasRole(inputs[0])
		case "hasanyrole":
			rolesResult = subject.HasAnyRole(inputs...)
		case "hasallroles":
			rolesResult = subject.HasAllRoles(inputs...)
		}
		if !rolesResult {
			reasons = append(reasons, &authz.Reason{
				Func:     fn,
				Expected: strings.Join(inputs, ", "),
				Got:      subject.AuthorizationInfo.Roles(),
			})
			break
		}
	}

	// Fail fast
	if !r.authorizationInfo.SatisfyEither() && !rolesResult {
		return false, reasons
	}

	// Check permissions
	permissionResult := true
	for fn, inputs := range r.authorizationInfo.Permissions {
		switch fn {
		case "ispermitted":
			permissionResult = subject.IsPermitted(inputs[0])
		case "ispermittedall":
			permissionResult = subject.IsPermittedAll(inputs...)
		}
		if !permissionResult {
			reasons = append(reasons, &authz.Reason{
				Func:     fn,
				Expected: strings.Join(inputs, ", "),
				Got:      subject.AuthorizationInfo.Permissions(),
			})
			break
		}
	}

	// Satisfy: either
	if r.authorizationInfo.SatisfyEither() {
		switch {
		case len(r.authorizationInfo.Roles) == 0:
			return permissionResult, reasons
		case len(r.authorizationInfo.Permissions) == 0:
			return rolesResult, reasons
		default:
			return rolesResult || permissionResult, reasons
		}
	}

	// Satisfy: both
	return rolesResult && permissionResult, reasons
}

// String method is Stringer interface.
func (r *Route) String() string {
	if r.IsStatic {
		if r.IsFile() {
			return fmt.Sprintf("staticroute(name:%s path:%s file:%s/%s)", r.Name, r.Path, r.Dir, r.File)
		}
		return fmt.Sprintf("staticroute(name:%s path:%s dir:%s listing:%v)", r.Name, r.Path, r.Dir, r.ListDir)
	}

	return fmt.Sprintf("route(name:%s method:%s path:%s target:%s.%s auth:%s maxbodysize:%v %s %v constraints(%v))",
		r.Name, r.Method, r.Path, r.Target, r.Action, r.Auth, r.MaxBodySize, r.CORS, r.authorizationInfo, r.Constraints)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported types and methods
//______________________________________________________________________________

type parentRouteInfo struct {
	AntiCSRFCheck     bool
	CORSEnabled       bool
	ParentName        string
	PrefixPath        string
	Target            string
	Auth              string
	MaxBodySizeStr    string
	CORS              *CORS
	AuthorizationInfo *authorizationInfo
}

type authorizationInfo struct {
	Satisfy     string
	Roles       map[string][]string
	Permissions map[string][]string
}

func (a *authorizationInfo) SatisfyEither() bool {
	return a.Satisfy == "either"
}

func (a *authorizationInfo) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString("authorizationinfo(satisfy:")
	buf.WriteString(a.Satisfy)

	buf.WriteString(" roles:[")
	for k, v := range a.Roles {
		buf.WriteString(k)
		buf.WriteByte('(')
		buf.WriteString(strings.Join(v, ","))
		buf.WriteString(") ")
	}
	buf.WriteByte(']')

	buf.WriteString(" permissions:[")
	for k, v := range a.Permissions {
		buf.WriteString(k)
		buf.WriteByte('(')
		buf.WriteString(strings.Join(v, "|"))
		buf.WriteString(") ")
	}
	buf.WriteString("])")

	return buf.String()
}

func parseAuthorizationInfo(cfg *config.Config, routeName string, parentRoute *parentRouteInfo) (*authorizationInfo, error) {
	info := &authorizationInfo{
		Satisfy: cfg.StringDefault(routeName+".authorization.satisfy", parentRoute.AuthorizationInfo.Satisfy),
	}

	roles, found := cfg.StringList(routeName + ".authorization.roles")
	if found && len(roles) > 0 {
		// roles = [
		//   "hasrole(manager)",
		//   "hasanyrole(role1, role2, role3)"
		// ]
		roles, err := parseAuthorizationValues(roles, ",", fmt.Sprintf("%v.authorization.roles", routeName))
		if err != nil {
			return nil, err
		}
		info.Roles = roles
	} else {
		info.Roles = parentRoute.AuthorizationInfo.Roles
	}

	permissions, found := cfg.StringList(routeName + ".authorization.permissions")
	if found && len(permissions) > 0 {
		// permissions = [
		//   "ispermitted(newsletter:read,write)",
		//   "ispermittedall(newsletter:read,write | newsletter:12345)"
		// ]
		permissions, err := parseAuthorizationValues(permissions, "|", fmt.Sprintf("%v.authorization.permissions", routeName))
		if err != nil {
			return nil, err
		}
		info.Permissions = permissions
	} else {
		info.Permissions = parentRoute.AuthorizationInfo.Permissions
	}

	// Check statisfy
	if info.Satisfy == "both" && (len(info.Roles) == 0 || len(info.Permissions) == 0) {
		return nil, fmt.Errorf("%v.authorization.satisfy configured as 'both', however roles and permissions is not configured",
			routeName)
	}

	return info, nil
}

func parseAuthorizationValues(srcValues []string, delim, errPrefix string) (map[string][]string, error) {
	info := make(map[string][]string)
	for pos, srcValue := range srcValues {
		// Check open and close brackets
		if strings.Count(srcValue, "(") != 1 || strings.Count(srcValue, ")") != 1 {
			return nil, fmt.Errorf("%v at index %v have incorrect open/close brackets '%v'",
				errPrefix, pos+1, srcValue)
		}

		start := strings.IndexByte(srcValue, '(')
		end := strings.IndexByte(srcValue, ')')

		// Parsing values
		var values []string
		for _, v := range strings.Split(srcValue[start+1:end], delim) {
			v = strings.TrimSpace(v)
			if len(v) > 0 {
				values = append(values, v)
			}
		}

		// Check values present
		if len(values) == 0 {
			return nil, fmt.Errorf("%v at index %v have func '%v' without input",
				errPrefix, pos+1, srcValue)
		}

		// Check input param count for certian methods
		fnName := srcValue[:start]
		if (fnName == "hasrole" || fnName == "ispermitted") && len(values) > 1 {
			return nil, fmt.Errorf("%v at index %v have func '%v' supports only one input param",
				errPrefix, pos+1, fnName)
		}

		info[fnName] = values
	}
	return info, nil
}

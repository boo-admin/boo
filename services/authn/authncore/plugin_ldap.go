package authncore

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/boo-admin/boo/client"
	ldap "github.com/go-ldap/ldap/v3"
	"github.com/mei-rune/iprange"
)

const (
	CfgUserLdapEnabled        = "users.ldap_enabled"
	CfgUserLdapAddress        = "users.ldap_address"
	CfgUserLdapTLS            = "users.ldap_tls"
	CfgUserLdapBaseDN         = "users.ldap_base_dn"
	CfgUserLdapFilter         = "users.ldap_filter"
	CfgUserLdapUserFormat     = "users.ldap_user_format"
	CfgUserLdapDefaultRoles   = "users.ldap_default_roles"
	CfgUserLdapLoginRoleField = "users.ldap_login_role_field"
	CfgUserLdapLoginRoleName  = "users.ldap_login_role"
)

type HasSource interface {
	Source() string
}

func isConnectError(err error) bool {
	if ldapErr, ok := err.(*ldap.Error); ok {
		if opErr, ok := ldapErr.Err.(*net.OpError); ok && opErr.Op == "dial" {
			return true
		}
	}
	return false
}

func LdapUserCheck(env *client.Environment, logger *slog.Logger) AuthOption {
	return AuthOptionFunc(func(auth *AuthService) error {
		ldapServer := env.Config.StringWithDefault(CfgUserLdapAddress, "")
		if ldapServer == "" {
			logger.Warn("ldap 没有配置，跳过它")
			return nil
		}
		ldapTLS := env.Config.BoolWithDefault(CfgUserLdapTLS, false)
		ldapDN := env.Config.StringWithDefault(CfgUserLdapBaseDN, "")
		ldapFilter := env.Config.StringWithDefault(CfgUserLdapFilter, "")
		ldapUserFormat := env.Config.StringWithDefault(CfgUserLdapUserFormat, "")
		if ldapUserFormat == "" {
			if ldapDN != "" {
				ldapUserFormat = "cn=%s," + ldapDN
			} else {
				ldapUserFormat = "%s"
			}
		}
		defaultRoles := strings.Split(env.Config.StringWithDefault(CfgUserLdapDefaultRoles, ""), ",")
		ldapRoles := env.Config.StringWithDefault(CfgUserLdapLoginRoleField,
			env.Config.StringWithDefault("users.ldap_roles", "memberOf"))
		exceptedRole := env.Config.StringWithDefault(CfgUserLdapLoginRoleName, "")

		auth.OnAuth(func(ctx *AuthContext) (bool, error) {
			isLdap := false
			isNew := false
			if ctx.Authentication != nil {
				u, ok := ctx.Authentication.(HasSource)
				if !ok {
					return false, nil
				}

				// if o := u.Data["source"]; o != nil {
				// 	method = strings.ToLower(fmt.Sprint(o))
				// }

				var method = u.Source()
				if method != "ldap" {
					return false, nil
				}
				isLdap = true
			} else {
				isNew = true
			}

			var l *ldap.Conn
			var err error
			if ldapTLS {
				l, err = ldap.DialTLS("tcp", ldapServer, &tls.Config{InsecureSkipVerify: true})
			} else {
				l, err = ldap.Dial("tcp", ldapServer)
			}
			if err != nil {
				logger.Info("尝试 LDAP 验证时，无法连接到 LDAP 服务器", slog.Any("error", err))

				if !isLdap {
					return false, nil
				}
				return isLdap, &ErrExternalServer{Msg: "无法连接到 LDAP 服务器" + err.Error()}
			}
			defer l.Close()

			username := fmt.Sprintf(ldapUserFormat, ctx.Request.Username)
			logger := ctx.Logger.With(
				slog.String("ldapServer", ldapServer),
				slog.Bool("ldapTLS", ldapTLS),
				slog.String("ldapDN", ldapDN),
				slog.String("ldapFilter", ldapFilter),
				slog.String("ldapUserFormat", ldapUserFormat),
			).With(slog.String("username", username), slog.String("password", "********"))
			// First bind with a read only user
			err = l.Bind(username, ctx.Request.Password)
			if err != nil {
				logger.Info("尝试 ldap 验证失败", slog.Any("error", err))
				if !isLdap {
					return false, nil
				}
				return isLdap, err
			}

			logger.Info("尝试 ldap 验证, 用户名和密码正确")

			if !isNew {
				if exceptedRole == "" {
					return true, nil
				}
			}

			var userRoles []string

			var ldapFilterForUser string
			if ldapFilter != "" {
				ldapFilterForUser = fmt.Sprintf(ldapFilter, username)
				if idx := strings.Index(username, "@"); idx > 0 {
					ldapFilterForUser = fmt.Sprintf(ldapFilter, username[:idx])
				}
			}

			// dn := "cn=" + username + "," + ldapDN
			//获取数据
			searchResult, err := l.Search(ldap.NewSearchRequest(
				ldapDN,
				ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
				ldapFilterForUser, nil, nil,
			))
			if err == nil {
				userRoles = make([]string, 0, 4)
				for _, ent := range searchResult.Entries {
					for _, attr := range ent.Attributes {
						if len(attr.Values) > 0 {
							if ldapRoles == attr.Name {
								for _, roleName := range attr.Values {
									dn, err := ldap.ParseDN(roleName)
									if err != nil {
										userRoles = append(userRoles, roleName)
										continue
									}

									if len(dn.RDNs) == 0 || len(dn.RDNs[0].Attributes) == 0 {
										continue
									}

									userRoles = append(userRoles, dn.RDNs[0].Attributes[0].Value)
								}
								// userData["roles"] = userRoles
								// userData["raw_roles"] = attr.Values
							}
							// userData[attr.Name] = attr.Values[0]
						}
					}
				}

				if exceptedRole != "" {
					found := false
					for _, role := range userRoles {
						if role == exceptedRole {
							found = true
							break
						}
					}

					if !found {
						if len(searchResult.Entries) == 0 {
							logger.Warn("user is permission denied - roles is empty", slog.String("exceptedRole", exceptedRole))
						} else {
							logger.Warn("user is permission denied", slog.String("exceptedRole", exceptedRole), slog.Any("roles", userRoles))
						}
						return true, ErrPermissionDenied
					}
				}
			} else {
				logger.Warn("search user and role fail", slog.Any("error", err))

				if exceptedRole != "" {
					return true, ErrPermissionDenied
				}
			}

			if isNew {
				ctx.Request.Username = strings.ToLower(ctx.Request.Username)
				ctx.Response.IsNewUser = true

				userInfo := &ldapUser{
					name:  ctx.Request.Username,
					roles: userRoles,
				}
				if len(defaultRoles) > 0 {
					roles := userInfo.RoleNames()
					for _, role := range defaultRoles {
						role = strings.TrimSpace(role)
						if role == "" {
							continue
						}
						roles = append(roles, role)
					}
					userInfo.roles = roles
				}
				ctx.Authentication = userInfo
				return true, nil
			}
			return true, nil
		})
		return nil
	})
}

var _ User = &ldapUser{}
var _ HasWhitelist = &ldapUser{}

type ldapUser struct {
	name  string
	roles []string
}

func (*ldapUser) IsLocked() bool {
	return false
}

func (*ldapUser) Source() string {
	return "ldap"
}

func (*ldapUser) IngressIPList() ([]iprange.Checker, error) {
	return nil, nil
}

func (u *ldapUser) RoleNames() []string {
	return u.roles
}

package rules

import (
	"github.com/haproxytech/client-native/v3/models"

	"github.com/haproxytech/kubernetes-ingress/pkg/haproxy/api"
	"github.com/haproxytech/kubernetes-ingress/pkg/utils"
)

type ReqSetVar struct {
	Name       string
	Scope      string
	Expression string
	CondTest   string
}

func (r ReqSetVar) GetType() Type {
	return REQ_SET_VAR
}

func (r ReqSetVar) Create(client api.HAProxyClient, frontend *models.Frontend, ingressACL string) error {
	if frontend.Mode == "tcp" {
		tcpRule := models.TCPRequestRule{
			Index:    utils.PtrInt64(0),
			Type:     "content",
			Action:   "set-var",
			VarName:  r.Name,
			VarScope: r.Scope,
			Expr:     r.Expression,
		}
		return client.FrontendTCPRequestRuleCreate(frontend.Name, tcpRule, ingressACL)
	}
	httpRule := models.HTTPRequestRule{
		Index:    utils.PtrInt64(0),
		Type:     "set-var",
		VarName:  r.Name,
		VarScope: r.Scope,
		VarExpr:  r.Expression,
	}
	if r.CondTest != "" {
		httpRule.Cond = "if"
		httpRule.CondTest = r.CondTest
	}
	return client.FrontendHTTPRequestRuleCreate(frontend.Name, httpRule, ingressACL)
}

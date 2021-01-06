// Copyright 2019 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"fmt"

	"github.com/haproxytech/kubernetes-ingress/controller/utils"
	"github.com/haproxytech/models/v2"
)

type FrontendHTTPReqs struct {
	Store    []models.HTTPRequestRule
	Presence map[uint64]struct{}
}
type FrontendHTTPRsps map[uint64]models.HTTPResponseRule

type FrontendTCPReqs struct {
	Store    []models.TCPRequestRule
	Presence map[uint64]struct{}
}

type Rule string

type rateLimitTable struct {
	size   *int64
	period *int64
}

const (
	//nolint
	BLACKLIST Rule = "blacklist"
	//nolint
	RATE_LIMIT Rule = "rate-limit"
	//nolint
	REQUEST_SET_HOST Rule = "set-host"
	//nolint
	SSL_REDIRECT Rule = "ssl-redirect"
	//nolint
	REQUEST_PATH_REWRITE Rule = "path-rewrite"
	//nolint
	PROXY_PROTOCOL Rule = "proxy-protocol"
	//nolint
	REQUEST_CAPTURE Rule = "request-capture"
	//nolint
	REQUEST_SET_HEADER Rule = "request-set-header"
	//nolint
	RESPONSE_SET_HEADER Rule = "response-set-header"
	//nolint
	WHITELIST Rule = "whitelist"
)

func NewFrontendTCPReqs() *FrontendTCPReqs {
	return &FrontendTCPReqs{
		Store:    make([]models.TCPRequestRule, 0),
		Presence: make(map[uint64]struct{}),
	}
}

func (frontendTCPReqs *FrontendTCPReqs) Add(key uint64, rule models.TCPRequestRule) {
	if _, ok := frontendTCPReqs.Presence[key]; ok {
		return
	}
	frontendTCPReqs.Presence[key] = struct{}{}
	frontendTCPReqs.Store = append([]models.TCPRequestRule{rule}, frontendTCPReqs.Store...)

}

func NewFrontendHTTPReqs() *FrontendHTTPReqs {
	return &FrontendHTTPReqs{
		Store:    make([]models.HTTPRequestRule, 0),
		Presence: make(map[uint64]struct{}),
	}
}

func (frontendHTTPReqs *FrontendHTTPReqs) Add(key uint64, rule models.HTTPRequestRule) {
	if _, ok := frontendHTTPReqs.Presence[key]; ok {
		return
	}
	frontendHTTPReqs.Presence[key] = struct{}{}
	frontendHTTPReqs.Store = append([]models.HTTPRequestRule{rule}, frontendHTTPReqs.Store...)

}

func (c *HAProxyController) FrontendHTTPRspsRefresh() (reload bool) {
	if c.cfg.FrontendRulesStatus[HTTP] == EMPTY {
		return false
	}

	// DELETE RULES
	c.Client.FrontendHTTPResponseRuleDeleteAll(FrontendHTTP)
	c.Client.FrontendHTTPResponseRuleDeleteAll(FrontendHTTPS)

	for _, frontend := range []string{FrontendHTTP, FrontendHTTPS} {
		// RESPONSE_SET_HEADER
		for _, httpRule := range c.cfg.FrontendHTTPRspRules[RESPONSE_SET_HEADER] {
			c.Logger.Error(c.Client.FrontendHTTPResponseRuleCreate(frontend, httpRule))
		}
	}
	return true
}

func (c *HAProxyController) FrontendHTTPReqsRefresh() (reload bool) {
	if c.cfg.FrontendRulesStatus[HTTP] == EMPTY {
		return false
	}

	c.Logger.Debug("Updating HTTP request rules for HTTP and HTTPS frontends")
	// DELETE RULES
	c.Client.FrontendHTTPRequestRuleDeleteAll(FrontendHTTP)
	c.Client.FrontendHTTPRequestRuleDeleteAll(FrontendHTTPS)
	//STATIC: FORWARDED_PRTOTO
	xforwardedprotoRule := models.HTTPRequestRule{
		Index:     utils.PtrInt64(0),
		Type:      "set-header",
		HdrName:   "X-Forwarded-Proto",
		HdrFormat: "https",
		Cond:      "if",
		CondTest:  "{ ssl_fc }",
	}
	c.Logger.Error(c.Client.FrontendHTTPRequestRuleCreate(FrontendHTTPS, xforwardedprotoRule))
	// SSL_REDIRECT
	for _, httpRule := range c.cfg.FrontendHTTPReqRules[SSL_REDIRECT].Store {
		c.Logger.Error(c.Client.FrontendHTTPRequestRuleCreate(FrontendHTTP, httpRule))
	}
	for _, frontend := range []string{FrontendHTTP, FrontendHTTPS} {
		// REQUEST_SET_HEADER
		for _, httpRule := range c.cfg.FrontendHTTPReqRules[REQUEST_SET_HEADER].Store {
			c.Logger.Error(c.Client.FrontendHTTPRequestRuleCreate(frontend, httpRule))
		}
		// REQUEST_CAPTURE
		for _, httpRule := range c.cfg.FrontendHTTPReqRules[REQUEST_CAPTURE].Store {
			c.Logger.Error(c.Client.FrontendHTTPRequestRuleCreate(frontend, httpRule))
		}
		// REQUEST_PATH_REWRITE
		for _, httpRule := range c.cfg.FrontendHTTPReqRules[REQUEST_PATH_REWRITE].Store {
			c.Logger.Error(c.Client.FrontendHTTPRequestRuleCreate(frontend, httpRule))
		}
		// REQUEST_SET_HOST
		for _, httpRule := range c.cfg.FrontendHTTPReqRules[REQUEST_SET_HOST].Store {
			c.Logger.Error(c.Client.FrontendHTTPRequestRuleCreate(frontend, httpRule))
		}
		// RATE_LIMIT
		for tableName, table := range rateLimitTables {
			_, err := c.Client.BackendGet(tableName)
			if err != nil {
				err := c.Client.BackendCreate(models.Backend{
					Name: tableName,
					StickTable: &models.BackendStickTable{
						Type:  "ip",
						Size:  table.size,
						Store: fmt.Sprintf("http_req_rate(%d)", *table.period),
					},
				})
				c.Logger.Error(err)
			}
		}
		for _, httpRule := range c.cfg.FrontendHTTPReqRules[RATE_LIMIT].Store {
			c.Logger.Error(c.Client.FrontendHTTPRequestRuleCreate(frontend, httpRule))
		}
		// BLACKLIST
		for _, httpRule := range c.cfg.FrontendHTTPReqRules[BLACKLIST].Store {
			c.Logger.Error(c.Client.FrontendHTTPRequestRuleCreate(frontend, httpRule))
		}
		// WHITELIST
		for _, httpRule := range c.cfg.FrontendHTTPReqRules[WHITELIST].Store {
			c.Logger.Error(c.Client.FrontendHTTPRequestRuleCreate(frontend, httpRule))
		}
		// STATIC: SET_VARIABLE txn.base (for logging purpose)
		setVarRule := models.HTTPRequestRule{
			Index:    utils.PtrInt64(0),
			Type:     "set-var",
			VarName:  "base",
			VarScope: "txn",
			VarExpr:  "base",
		}
		c.Logger.Error(c.Client.FrontendHTTPRequestRuleCreate(frontend, setVarRule))
		// STATIC: SET_VARIABLE txn.path (to use in http rules)
		setVarRule = models.HTTPRequestRule{
			Index:    utils.PtrInt64(0),
			Type:     "set-var",
			VarName:  "path",
			VarScope: "txn",
			VarExpr:  "path",
		}
		c.Logger.Error(c.Client.FrontendHTTPRequestRuleCreate(frontend, setVarRule))
		// STATIC: SET_VARIABLE txn.host (to use in http rules)
		setVarRule = models.HTTPRequestRule{
			Index:    utils.PtrInt64(0),
			Type:     "set-var",
			VarName:  "host",
			VarScope: "txn",
			VarExpr:  "req.hdr(Host),field(1,:),lower",
		}
		c.Logger.Error(c.Client.FrontendHTTPRequestRuleCreate(frontend, setVarRule))
	}
	return true
}

func (c *HAProxyController) FrontendTCPreqsRefresh() (reload bool) {
	if c.cfg.FrontendRulesStatus[TCP] == EMPTY {
		return false
	}
	c.Logger.Debug("Updating TCP request rules for HTTP and HTTPS frontends")
	// HTTP and HTTPS Frrontends
	for _, frontend := range []string{FrontendHTTP, FrontendHTTPS} {
		// DELETE RULES
		c.Client.FrontendTCPRequestRuleDeleteAll(frontend)
	}
	// PROXY_PROTCOL
	if len(c.cfg.FrontendTCPRules[PROXY_PROTOCOL].Store) > 0 {
		c.Logger.Error(c.Client.FrontendTCPRequestRuleCreate(FrontendHTTP, c.cfg.FrontendTCPRules[PROXY_PROTOCOL].Store[0]))
		if !c.cfg.SSLPassthrough {
			c.Logger.Error(c.Client.FrontendTCPRequestRuleCreate(FrontendHTTPS, c.cfg.FrontendTCPRules[PROXY_PROTOCOL].Store[0]))
		}
	}
	if !c.cfg.SSLPassthrough {
		return true
	}

	// SSL Frontend for SSL_PASSTHROUGH
	c.Client.FrontendTCPRequestRuleDeleteAll(FrontendSSL)
	// STATIC: Accept content
	err := c.Client.FrontendTCPRequestRuleCreate(FrontendSSL, models.TCPRequestRule{
		Index:    utils.PtrInt64(0),
		Action:   "accept",
		Type:     "content",
		Cond:     "if",
		CondTest: "{ req_ssl_hello_type 1 }",
	})
	c.Logger.Error(err)
	// REQUEST_CAPTURE
	for _, tcpRule := range c.cfg.FrontendTCPRules[REQUEST_CAPTURE].Store {
		c.Logger.Error(c.Client.FrontendTCPRequestRuleCreate(FrontendSSL, tcpRule))
	}
	// STATIC: Set-var rule used to log SNI
	err = c.Client.FrontendTCPRequestRuleCreate(FrontendSSL, models.TCPRequestRule{
		Index:    utils.PtrInt64(0),
		Action:   "set-var",
		VarName:  "sni",
		VarScope: "sess",
		Expr:     "req_ssl_sni",
		Type:     "content",
	})
	c.Logger.Error(err)
	// STATIC: Inspect delay
	inspectTimeout := utils.PtrInt64(5000)
	annTimeout, _ := GetValueFromAnnotations("timeout-client", c.cfg.ConfigMap.Annotations)
	if annTimeout != nil {
		if value, errParse := utils.ParseTime(annTimeout.Value); errParse == nil {
			inspectTimeout = value
		} else {
			c.Logger.Error(errParse)
		}
	}
	err = c.Client.FrontendTCPRequestRuleCreate(FrontendSSL, models.TCPRequestRule{
		Type:    "inspect-delay",
		Index:   utils.PtrInt64(0),
		Timeout: inspectTimeout,
	})
	c.Logger.Error(err)
	// BLACKLIST
	for _, tcpRule := range c.cfg.FrontendTCPRules[BLACKLIST].Store {
		c.Logger.Error(c.Client.FrontendTCPRequestRuleCreate(FrontendSSL, tcpRule))
	}
	// WHITELIST
	for _, tcpRule := range c.cfg.FrontendTCPRules[WHITELIST].Store {
		c.Logger.Error(c.Client.FrontendTCPRequestRuleCreate(FrontendSSL, tcpRule))
	}
	// PROXY_PROTCOL
	if len(c.cfg.FrontendTCPRules[PROXY_PROTOCOL].Store) > 0 {
		c.Logger.Error(c.Client.FrontendTCPRequestRuleCreate(FrontendSSL, c.cfg.FrontendTCPRules[PROXY_PROTOCOL].Store[0]))
	}
	return true
}

/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package certmod

import (
	"github.com/sprintframework/cert"
	"github.com/sprintframework/certmod/netlify"
	"github.com/sprintframework/sprint"
)

type certScanner struct {
	Scan     []interface{}
}

func Scanner(scan... interface{}) sprint.CoreScanner {
	return &certScanner {
		Scan: scan,
	}
}

func (t *certScanner) CoreBeans() []interface{} {

	beans := []interface{}{
		CertificateIssueService(),
		CertificateRepository(),
		CertificateService(),
		CertificateManager(),
		netlify.NetlifyChallenge(),
		&struct {
			DNSChallenges []cert.DNSChallenge `inject`
		}{},
		DynDNSService(),
	}

	return append(beans, t.Scan...)
}


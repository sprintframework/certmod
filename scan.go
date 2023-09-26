/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package certmod

import (
	"github.com/sprintframework/certmod/netlify"
)

var CertServices = []interface{} {
	CertificateIssueService(),
	CertificateRepository(),
	CertificateService(),
	CertificateManager(),
	netlify.NetlifyChallenge(),
	DynDNSService(),
}



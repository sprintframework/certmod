/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package certmod

import (
	"github.com/sprintframework/sprint"
)

type certScanner struct {
	Scan     []interface{}
}

func CoreScanner(scan... interface{}) sprint.CoreScanner {
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
	}

	return append(beans, t.Scan...)
}


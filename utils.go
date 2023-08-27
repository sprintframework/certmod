/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package certmod

import (
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"strings"
)

func indexStrings(a []string) map[string]bool {
	m := make(map[string]bool)
	for _, part := range a {
		key := strings.TrimSpace(part)
		m[key] = true
	}
	return m
}

func asList(m map[string]bool) []string {
	var a []string
	for k := range m {
		a = append(a, k)
	}
	return a
}

func getKeyType(algorithm string) certcrypto.KeyType {
	switch strings.ToUpper(algorithm) {
	case "RSA2048":
		return certcrypto.RSA2048
	case "RSA4096":
		return certcrypto.RSA4096
	case "RSA8192":
		return certcrypto.RSA8192
	case "EC256":
		return certcrypto.EC256
	case "EC384":
		return certcrypto.EC384
	}
	return certcrypto.RSA2048
}

func ToZone(domain string) (string, error) {

	fqdn := ToFqdn(domain)
	zone, err := dns01.FindZoneByFqdn(fqdn)
	if err != nil {
		return "", err
	}

	return UnFqdn(zone), nil
}

// ToFqdn converts the name into a fqdn appending a trailing dot.
func ToFqdn(name string) string {
	n := len(name)
	if n == 0 || name[n-1] == '.' {
		return name
	}
	return name + "."
}

// UnFqdn converts the fqdn into a name removing the trailing dot.
func UnFqdn(name string) string {
	n := len(name)
	if n != 0 && name[n-1] == '.' {
		return name[:n-1]
	}
	return name
}


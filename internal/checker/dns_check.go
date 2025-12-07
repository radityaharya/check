package checker

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"gocheck/internal/models"
)

func (e *Engine) performDNSCheck(check *models.Check, history *models.CheckHistory, start time.Time) {
	if check.DNSHostname == "" {
		history.Success = false
		history.ErrorMessage = "no hostname specified"
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}

	recordType := check.DNSRecordType
	if recordType == "" {
		recordType = "A"
	}

	resolver := &net.Resolver{
		PreferGo: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(check.TimeoutSeconds)*time.Second)
	defer cancel()

	var records []string
	var err error

	switch strings.ToUpper(recordType) {
	case "A":
		var ips []net.IP
		ips, err = resolver.LookupIP(ctx, "ip4", check.DNSHostname)
		for _, ip := range ips {
			records = append(records, ip.String())
		}
	case "AAAA":
		var ips []net.IP
		ips, err = resolver.LookupIP(ctx, "ip6", check.DNSHostname)
		for _, ip := range ips {
			records = append(records, ip.String())
		}
	case "CNAME":
		var cname string
		cname, err = resolver.LookupCNAME(ctx, check.DNSHostname)
		if err == nil {
			records = append(records, cname)
		}
	case "MX":
		var mxs []*net.MX
		mxs, err = resolver.LookupMX(ctx, check.DNSHostname)
		if err == nil {
			for _, mx := range mxs {
				records = append(records, fmt.Sprintf("%s (priority: %d)", mx.Host, mx.Pref))
			}
		}
	case "TXT":
		var txts []string
		txts, err = resolver.LookupTXT(ctx, check.DNSHostname)
		if err == nil {
			records = txts
		}
	default:
		err = fmt.Errorf("unsupported record type: %s", recordType)
	}

	history.ResponseTimeMs = int(time.Since(start).Milliseconds())

	if err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("DNS lookup failed: %v", err)
		return
	}

	if len(records) == 0 {
		history.Success = false
		history.ErrorMessage = "no records found"
		return
	}

	history.ResponseBody = strings.Join(records, ", ")

	if check.ExpectedDNSValue != "" {
		found := false
		for _, record := range records {
			if record == check.ExpectedDNSValue || strings.Contains(record, check.ExpectedDNSValue) {
				found = true
				break
			}
		}
		if found {
			history.Success = true
		} else {
			history.Success = false
			history.ErrorMessage = fmt.Sprintf("expected value '%s' not found in records: %v", check.ExpectedDNSValue, records)
		}
	} else {
		history.Success = true
	}
}

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"gocheck/proto/pb"

	_ "github.com/lib/pq"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	region := flag.String("region", os.Getenv("REGION"), "Region code (e.g., us-east-1)")
	token := flag.String("token", os.Getenv("PROBE_TOKEN"), "Probe authentication token")
	serverAddr := flag.String("server", os.Getenv("SENTINEL_ADDR"), "Sentinel server address (e.g., localhost:50051)")
	flag.Parse()

	if *region == "" {
		log.Fatal("Region code is required (use -region flag or REGION env var)")
	}
	if *token == "" {
		log.Fatal("Probe token is required (use -token flag or PROBE_TOKEN env var)")
	}
	if *serverAddr == "" {
		*serverAddr = "localhost:50051"
	}

	for {
		if err := connectAndListen(*region, *token, *serverAddr); err != nil {
			log.Printf("Connection error: %v, reconnecting in 2 seconds...", err)
			time.Sleep(2 * time.Second)
		}
	}
}

func connectAndListen(region, token, serverAddr string) error {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	client := pb.NewSentinelClient(conn)
	ctx := context.Background()
	stream, err := client.EstablishConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to establish connection: %w", err)
	}

	err = stream.Send(&pb.ProbeMessage{
		Payload: &pb.ProbeMessage_Register{
			Register: &pb.Register{
				RegionCode: region,
				Token:      token,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	log.Printf("Connected to Sentinel at %s as region %s", serverAddr, region)

	go sendHeartbeats(stream)

	for {
		cmd, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("failed to receive command: %w", err)
		}

		if cmd.GetCommandType() == "CHECK_NOW" {
			log.Printf("[CHECK_NOW] Received check request for check_id=%d, type=%s", cmd.GetCheckId(), cmd.GetCheckType())
			go performCheck(stream, cmd, region)
		} else {
			log.Printf("[COMMAND] Received unknown command: %s", cmd.GetCommandType())
		}
	}
}

func sendHeartbeats(stream pb.Sentinel_EstablishConnectionClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		err := stream.Send(&pb.ProbeMessage{
			Payload: &pb.ProbeMessage_Heartbeat{
				Heartbeat: &pb.Heartbeat{
					Timestamp: time.Now().Unix(),
				},
			},
		})
		if err != nil {
			log.Printf("Failed to send heartbeat: %v", err)
			return
		}
	}
}

func performCheck(stream pb.Sentinel_EstablishConnectionClient, cmd *pb.ServerCommand, region string) {
	timeoutSeconds := int(cmd.GetTimeoutSeconds())
	if timeoutSeconds == 0 {
		timeoutSeconds = 10
	}

	start := time.Now()
	var success bool
	var statusCode int32
	var errorMessage string

	checkType := cmd.GetCheckType()
	if checkType == "" && cmd.GetUrl() != "" {
		checkType = "http"
	}

	log.Printf("[CHECK] Starting %s check for check_id=%d, region=%s", checkType, cmd.GetCheckId(), region)

	switch checkType {
	case "http", "json_http":
		success, statusCode, errorMessage = performHTTPCheck(cmd, timeoutSeconds)
	case "ping":
		success, statusCode, errorMessage = performPingCheck(cmd, timeoutSeconds)
	case "postgres":
		success, statusCode, errorMessage = performPostgresCheck(cmd, timeoutSeconds)
	case "dns":
		success, statusCode, errorMessage = performDNSCheck(cmd, timeoutSeconds)
	default:
		success = false
		statusCode = 0
		errorMessage = fmt.Sprintf("unsupported check type: %s", checkType)
	}

	latency := int32(time.Since(start).Milliseconds())

	if success {
		log.Printf("[CHECK] Check completed successfully for check_id=%d, type=%s, latency=%dms", cmd.GetCheckId(), checkType, latency)
	} else {
		log.Printf("[CHECK] Check failed for check_id=%d, type=%s, error=%s, latency=%dms", cmd.GetCheckId(), checkType, errorMessage, latency)
	}

	result := &pb.CheckResult{
		CheckId:      cmd.GetCheckId(),
		Region:      region,
		LatencyMs:   latency,
		Success:     success,
		StatusCode:  statusCode,
		ErrorMessage: errorMessage,
	}

	err := stream.Send(&pb.ProbeMessage{
		Payload: &pb.ProbeMessage_Result{
			Result: result,
		},
	})
	if err != nil {
		log.Printf("Failed to send check result: %v", err)
	}
}

func performHTTPCheck(cmd *pb.ServerCommand, timeoutSeconds int) (bool, int32, string) {
	if cmd.GetUrl() == "" {
		return false, 0, "no URL specified"
	}

	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}

	method := cmd.GetMethod()
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequest(method, cmd.GetUrl(), nil)
	if err != nil {
		return false, 0, fmt.Sprintf("invalid request: %v", err)
	}

	if cmd.GetCheckType() == "json_http" {
		req.Header.Set("Accept", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, 0, err.Error()
	}
	defer resp.Body.Close()

	statusCode := int32(resp.StatusCode)
	success := resp.StatusCode >= 200 && resp.StatusCode < 400

	if cmd.GetCheckType() == "json_http" && success && cmd.GetJsonPath() != "" {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, statusCode, fmt.Sprintf("failed to read body: %v", err)
		}

		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			return false, statusCode, fmt.Sprintf("invalid JSON: %v", err)
		}

		value, err := extractJSONValue(jsonData, cmd.GetJsonPath())
		if err != nil {
			return false, statusCode, fmt.Sprintf("JSON path error: %v", err)
		}

		if cmd.GetExpectedJsonValue() != "" {
			valueStr := fmt.Sprintf("%v", value)
			if valueStr != cmd.GetExpectedJsonValue() {
				return false, statusCode, fmt.Sprintf("expected '%s', got '%s'", cmd.GetExpectedJsonValue(), valueStr)
			}
		}
	}

	if !success {
		return false, statusCode, fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
	}

	return true, statusCode, ""
}

func performPingCheck(cmd *pb.ServerCommand, timeoutSeconds int) (bool, int32, string) {
	host := cmd.GetHost()
	if host == "" {
		return false, 0, "no host specified"
	}

	timeout := time.Duration(timeoutSeconds) * time.Second
	var cmdExec *exec.Cmd

	if runtime.GOOS == "windows" {
		cmdExec = exec.Command("ping", "-n", "1", "-w", fmt.Sprintf("%d", timeoutSeconds*1000), host)
	} else {
		cmdExec = exec.Command("ping", "-c", "1", "-W", fmt.Sprintf("%d", timeoutSeconds), host)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmdExec = exec.CommandContext(ctx, cmdExec.Path, cmdExec.Args[1:]...)

	output, err := cmdExec.CombinedOutput()
	if err != nil {
		return false, 0, fmt.Sprintf("ping failed: %v", err)
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "time=") || strings.Contains(outputStr, "Time=") {
		return true, 200, ""
	} else if strings.Contains(outputStr, "bytes from") || strings.Contains(outputStr, "Reply from") {
		return true, 200, ""
	}

	return false, 0, "no response from host"
}

func performPostgresCheck(cmd *pb.ServerCommand, timeoutSeconds int) (bool, int32, string) {
	if cmd.GetPostgresConnString() == "" {
		return false, 0, "no connection string specified"
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", cmd.GetPostgresConnString())
	if err != nil {
		return false, 0, fmt.Sprintf("connection error: %v", err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(time.Duration(timeoutSeconds) * time.Second)
	db.SetMaxOpenConns(1)

	if cmd.GetPostgresQuery() == "" {
		err = db.PingContext(ctx)
		if err != nil {
			return false, 0, fmt.Sprintf("ping failed: %v", err)
		}
		return true, 200, ""
	}

	var result string
	err = db.QueryRowContext(ctx, cmd.GetPostgresQuery()).Scan(&result)
	if err != nil {
		return false, 0, fmt.Sprintf("query failed: %v", err)
	}

	if cmd.GetExpectedQueryValue() != "" {
		if result != cmd.GetExpectedQueryValue() {
			return false, 200, fmt.Sprintf("expected '%s', got '%s'", cmd.GetExpectedQueryValue(), result)
		}
	}

	return true, 200, ""
}

func performDNSCheck(cmd *pb.ServerCommand, timeoutSeconds int) (bool, int32, string) {
	if cmd.GetDnsHostname() == "" {
		return false, 0, "no hostname specified"
	}

	recordType := cmd.GetDnsRecordType()
	if recordType == "" {
		recordType = "A"
	}

	resolver := &net.Resolver{
		PreferGo: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	var records []string
	var err error

	switch strings.ToUpper(recordType) {
	case "A":
		var ips []net.IP
		ips, err = resolver.LookupIP(ctx, "ip4", cmd.GetDnsHostname())
		for _, ip := range ips {
			records = append(records, ip.String())
		}
	case "AAAA":
		var ips []net.IP
		ips, err = resolver.LookupIP(ctx, "ip6", cmd.GetDnsHostname())
		for _, ip := range ips {
			records = append(records, ip.String())
		}
	case "CNAME":
		var cname string
		cname, err = resolver.LookupCNAME(ctx, cmd.GetDnsHostname())
		if err == nil {
			records = append(records, cname)
		}
	case "MX":
		var mxs []*net.MX
		mxs, err = resolver.LookupMX(ctx, cmd.GetDnsHostname())
		if err == nil {
			for _, mx := range mxs {
				records = append(records, fmt.Sprintf("%s (priority: %d)", mx.Host, mx.Pref))
			}
		}
	case "TXT":
		var txts []string
		txts, err = resolver.LookupTXT(ctx, cmd.GetDnsHostname())
		if err == nil {
			records = txts
		}
	default:
		err = fmt.Errorf("unsupported record type: %s", recordType)
	}

	if err != nil {
		return false, 0, fmt.Sprintf("DNS lookup failed: %v", err)
	}

	if len(records) == 0 {
		return false, 0, "no records found"
	}

	if cmd.GetExpectedDnsValue() != "" {
		found := false
		for _, record := range records {
			if record == cmd.GetExpectedDnsValue() || strings.Contains(record, cmd.GetExpectedDnsValue()) {
				found = true
				break
			}
		}
		if !found {
			return false, 200, fmt.Sprintf("expected value '%s' not found in records: %v", cmd.GetExpectedDnsValue(), records)
		}
	}

	return true, 200, ""
}

func extractJSONValue(data interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil, fmt.Errorf("key '%s' not found", part)
			}
		case []interface{}:
			idx := 0
			if _, err := fmt.Sscanf(part, "%d", &idx); err != nil {
				return nil, fmt.Errorf("invalid array index: %s", part)
			}
			if idx < 0 || idx >= len(v) {
				return nil, fmt.Errorf("array index %d out of bounds", idx)
			}
			current = v[idx]
		default:
			return nil, fmt.Errorf("cannot access '%s' on non-object/non-array", part)
		}
	}

	return current, nil
}


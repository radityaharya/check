package grpc_server

import (
	"log"
	"sync"
	"time"

	"gocheck/internal/db"
	"gocheck/internal/models"
	"gocheck/proto/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SentinelServer struct {
	pb.UnimplementedSentinelServer
	db       *db.Database
	registry sync.Map
	engine   interface {
		BroadcastCheckResult(check models.Check, history *models.CheckHistory)
	}
}

func NewSentinelServer(database *db.Database) *SentinelServer {
	return &SentinelServer{
		db: database,
	}
}

func NewSentinelServerWithEngine(database *db.Database, engine interface {
	BroadcastCheckResult(check models.Check, history *models.CheckHistory)
}) *SentinelServer {
	return &SentinelServer{
		db:     database,
		engine: engine,
	}
}

func (s *SentinelServer) EstablishConnection(stream pb.Sentinel_EstablishConnectionServer) error {
	var region string
	var probeID int64

	for {
		msg, err := stream.Recv()
		if err != nil {
			s.disconnect(region, probeID)
			return err
		}

		switch payload := msg.Payload.(type) {
		case *pb.ProbeMessage_Register:
			probeID, err = s.handleRegister(payload.Register, stream)
			if err != nil {
				return err
			}
			region = payload.Register.RegionCode
			s.registry.Store(region, stream)
			log.Printf("Probe connected: %s (ID: %d)", region, probeID)

		case *pb.ProbeMessage_Result:
			if probeID == 0 {
				return status.Error(codes.Unauthenticated, "not registered")
			}
			err = s.handleCheckResult(probeID, region, payload.Result)
			if err != nil {
				log.Printf("Failed to save check result: %v", err)
			}

		case *pb.ProbeMessage_Heartbeat:
			if probeID == 0 {
				return status.Error(codes.Unauthenticated, "not registered")
			}
			err = s.db.UpdateProbeLastSeen(probeID)
			if err != nil {
				log.Printf("Failed to update probe last seen: %v", err)
			}
		}
	}
}

func (s *SentinelServer) handleRegister(reg *pb.Register, stream pb.Sentinel_EstablishConnectionServer) (int64, error) {
	probeID, err := s.db.ValidateProbeToken(reg.Token)
	if err != nil {
		return 0, status.Error(codes.Unauthenticated, "invalid token")
	}

	err = s.db.UpdateProbeStatus(probeID, "ONLINE")
	if err != nil {
		log.Printf("Failed to update probe status: %v", err)
	}

	return probeID, nil
}

func (s *SentinelServer) handleCheckResult(probeID int64, region string, result *pb.CheckResult) error {
	history := &models.CheckHistory{
		CheckID:        result.CheckId,
		StatusCode:     int(result.StatusCode),
		ResponseTimeMs: int(result.LatencyMs),
		Success:        result.Success,
		ErrorMessage:   result.ErrorMessage,
		CheckedAt:      time.Now().UTC(),
		ProbeID:        &probeID,
		Region:         region,
	}

	log.Printf("[PROBE] Received check result: check_id=%d, region=%s, success=%v, latency=%dms", result.CheckId, region, result.Success, result.LatencyMs)

	err := s.db.AddHistory(history)
	if err != nil {
		return err
	}

	// Broadcast to SSE clients if engine is available
	if s.engine != nil {
		check, err := s.db.GetCheck(result.CheckId)
		if err == nil {
			s.engine.BroadcastCheckResult(*check, history)
		} else {
			log.Printf("Failed to get check %d for SSE broadcast: %v", result.CheckId, err)
		}
	}

	return nil
}

func (s *SentinelServer) disconnect(region string, probeID int64) {
	if region != "" {
		s.registry.Delete(region)
		log.Printf("Probe disconnected: %s", region)
	}
	if probeID != 0 {
		err := s.db.UpdateProbeStatus(probeID, "OFFLINE")
		if err != nil {
			log.Printf("Failed to update probe status on disconnect: %v", err)
		}
	}
}

func (s *SentinelServer) BroadcastCheckFull(check models.Check) {
	s.BroadcastCheckToRegion(check, "")
}

func (s *SentinelServer) BroadcastCheckToRegion(check models.Check, region string) {
	timeoutSeconds := int32(check.TimeoutSeconds)
	if timeoutSeconds == 0 {
		timeoutSeconds = 10
	}

	cmd := &pb.ServerCommand{
		CommandType:        "CHECK_NOW",
		CheckId:            check.ID,
		CheckType:          string(check.Type),
		Url:                check.URL,
		Host:               check.Host,
		PostgresConnString: check.PostgresConnString,
		PostgresQuery:      check.PostgresQuery,
		ExpectedQueryValue: check.ExpectedQueryValue,
		DnsHostname:        check.DNSHostname,
		DnsRecordType:      check.DNSRecordType,
		ExpectedDnsValue:   check.ExpectedDNSValue,
		Method:             check.Method,
		TimeoutSeconds:     timeoutSeconds,
		JsonPath:           check.JSONPath,
		ExpectedJsonValue:  check.ExpectedJSONValue,
	}

	if region != "" {
		if stream, ok := s.registry.Load(region); ok {
			if err := stream.(pb.Sentinel_EstablishConnectionServer).Send(cmd); err != nil {
				log.Printf("Failed to send command to probe %s: %v", region, err)
				s.registry.Delete(region)
			} else {
				log.Printf("Triggered check %d for region %s", check.ID, region)
			}
		} else {
			log.Printf("No probe connected for region %s", region)
		}
		return
	}

	s.registry.Range(func(key, value interface{}) bool {
		stream := value.(pb.Sentinel_EstablishConnectionServer)
		if err := stream.Send(cmd); err != nil {
			log.Printf("Failed to send command to probe %v: %v", key, err)
			s.registry.Delete(key)
		}
		return true
	})
}


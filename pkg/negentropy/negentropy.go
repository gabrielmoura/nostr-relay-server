package negentropy

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/goccy/go-json"
	"github.com/illuzen/go-negentropy"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"strconv"
)

const (
	NegentropyProtocol = "/negentropy/1.0.0"
	FrameSizeLimit     = 1024 * 1024
	IdSize             = 32
)

func HandleNegOpen(ws *dto.WsServer, data dto.Data) error {
	var payloadHex string
	json.Unmarshal(data[3], &payloadHex)

	filterByte := data[2]

	var filter nostr.Filter

	err := json.Unmarshal(filterByte, &filter)
	if err != nil {
		log.Logger.Error("Error unmarshaling filter", zap.Error(err))
		return err
	}

	events, err := db.DbQueries.QueryEvents(context.Background(), filter)
	if err != nil {
		return err
	}
	//logging.Infof("%s has %d events", hostId, len(events))
	log.Logger.Info("Negentropy events count",
		//zap.String("hostId", data[1]),
		zap.Int("eventsCount", len(events)),
	)

	vector, err := LoadEventVector(events)
	if err != nil {
		return err
	}

	//logging.Infof("%s sealed the events", hostId)
	log.Logger.Info("Negentropy vector sealed",
		zap.String("hostId", string(data[1])),
	)
	// intentional shadowing
	neg, err := negentropy.NewNegentropy(vector, FrameSizeLimit)
	if err != nil {
		return err
	}

	log.Logger.Debug("Negentropy payload hex",
		zap.String("payloadHex", string(payloadHex)),
	)
	decodedBytes, err := hex.DecodeString(string(payloadHex))
	if err != nil {
		return fmt.Errorf("failed to decode hex string: %w", err)
	}
	msg, err := neg.Reconcile(decodedBytes)
	if err != nil {
		return err
	}

	nMsg, _ := BuildNegentropyMessage(dto.TypeNegMsg, filter, msg, "", []string{}, []byte{})
	ws.ChanSender <- nMsg
	return nil
}
func HandleNegNeed(ws *dto.WsServer, data dto.Data) error {
	var needIds []string

	err := json.Unmarshal(data[3], &needIds)
	if err != nil {
		return fmt.Errorf("failed to unmarshal need IDs: %w", err)
	}

	haveEvents, err := db.DbQueries.QueryEvents(context.Background(), nostr.Filter{
		IDs: needIds,
	})
	if err != nil {
		return err
	}
	haveBytes, err := json.Marshal(haveEvents)
	if err != nil {
		//logging.Infof("Error marshaling to JSON:%s", err)
		log.Logger.Error("Error marshaling to JSON",
			zap.Error(err),
		)
		return err
	}
	msgHave, err := BuildNegentropyMessage(dto.TypeNegHave, nostr.Filter{}, []byte{}, "", []string{}, haveBytes)
	if err != nil {
		log.Logger.Error("Error building Negentropy message",
			zap.Error(err),
		)
		return err
	}
	ws.ChanSender <- msgHave
	return nil
}
func HandleNegHave(ws *dto.WsServer, data dto.Data) error {
	var newEvents []*nostr.Event
	err := json.Unmarshal(data[3], &newEvents)
	if err != nil {
		return err
	}

	for _, event := range newEvents {
		err := db.DbQueries.InsertEvent(context.Background(), event)
		if err != nil {
			log.Logger.Error("Could not store event",
				zap.String("eventId", event.ID),
			)
			continue
		}
		// TODO: this needs to be more thoroughly tested
		//if event.Kind == 117 {
		//	// do leaf sync
		//	root, found := GetScionicRoot(event)
		//	if !found {
		//		logging.Infof("Event of type 117 with no 'scionic_root' tag, skipping tree download %+v", event)
		//		continue
		//	}
		//
		//	DownloadDag(root)
		//}
	}
	//if final {
	//	msgClose, err := BuildNegentropyMessage(dto.TypeNegClose, nostr.Filter{}, []byte{}, "", []string{}, []byte{})
	//	if err != nil {
	//		log.Logger.Error("Error building Negentropy message",
	//			zap.Error(err),
	//		)
	//		return err
	//	}
	//	ws.ChanSender <- msgClose
	//}

	return nil
}
func HandleNegMsg(ws *dto.WsServer, data dto.Data) error {
	var payloadHex string
	json.Unmarshal(data[2], &payloadHex)
	initiator := false

	var msg []byte
	var have, need []string

	var filter nostr.Filter

	decodedBytes, err := hex.DecodeString(payloadHex)
	if err != nil {
		return fmt.Errorf("failed to decode hex string: %w", err)
	}

	events, err := db.DbQueries.QueryEvents(context.Background(), filter)
	if err != nil {
		return err
	}
	//logging.Infof("%s has %d events", hostId, len(events))
	log.Logger.Info("Negentropy events count",
		//zap.String("hostId", data[1]),
		zap.Int("eventsCount", len(events)),
	)

	vector, err := LoadEventVector(events)
	if err != nil {
		return err
	}

	//logging.Infof("%s sealed the events", hostId)
	log.Logger.Info("Negentropy vector sealed",
		zap.String("hostId", string(data[1])),
	)

	neg, err := negentropy.NewNegentropy(vector, 1024*1024)
	if err != nil {
		return err
	}
	if initiator {
		msg, err = neg.ReconcileWithIDs(decodedBytes, &have, &need)
		if err != nil {
			return err
		}

		log.Logger.Info("Negentropy reconcile",
			zap.String("hostId", string(data[1])),
			zap.Int("haveCount", len(have)),
			zap.Int("needCount", len(need)),
		)

		// upload have
		if len(have) > 0 {
			hexHave := make([]string, len(have))
			for i, s := range have {
				hexHave[i] = hex.EncodeToString([]byte(s))
			}

			filter := nostr.Filter{
				IDs: hexHave,
			}

			haveEvents, err := db.DbQueries.QueryEvents(context.Background(), filter)
			if err != nil {
				return err
			}
			//logging.Info(haveEvents)

			// Marshal the array of events to JSON
			haveBytes, err := json.Marshal(haveEvents)
			if err != nil {
				//logging.Infof("Error marshaling to JSON:%s", err)
				log.Logger.Error("Error marshaling to JSON",
					zap.Error(err),
				)
				return err
			}

			// upload
			msgHave, err := BuildNegentropyMessage(dto.TypeNegHave, filter, []byte{}, "", []string{}, haveBytes)
			if err != nil {
				log.Logger.Error("Error building Negentropy message",
					zap.Error(err),
				)
				return err
			}
			ws.ChanSender <- msgHave
		}

		// download need if needed
		if len(need) > 0 {
			needIds := make([]string, len(need))
			for i, s := range need {
				needIds[i] = hex.EncodeToString([]byte(s))
			}
			//if err = SendNegentropyMessage(hostId, stream, "NEG-NEED", nostr.Filter{}, []byte{}, "", needIds, []byte{}); err != nil {
			//	return err
			//}
			msgNeed, err := BuildNegentropyMessage(dto.TypeNegNeed, filter, []byte{}, "", needIds, []byte{})
			if err != nil {
				log.Logger.Error("Error building Negentropy message",
					zap.Error(err),
				)
				return err
			}
			ws.ChanSender <- msgNeed
		}
	} else {
		msg, err = neg.Reconcile(decodedBytes)
	}
	if err != nil {
		return err
	}

	if len(msg) == 0 {
		//logging.Infof(hostId, "%s: Sync complete")
		log.Logger.Info("Negentropy sync complete")
		if len(need) == 0 {
			// we are done
			//if err = SendNegentropyMessage(hostId, stream, "NEG-CLOSE", nostr.Filter{}, []byte{}, "", []string{}, []byte{}); err != nil {
			//	return err
			//}
			msgClose, err := BuildNegentropyMessage(dto.TypeNegClose, nostr.Filter{}, []byte{}, "", []string{}, []byte{})
			if err != nil {
				log.Logger.Error("Error building Negentropy message",
					zap.Error(err),
				)
				return err
			}
			ws.ChanSender <- msgClose
			return nil
		}
	} else {
		//logging.Infof(hostId, "%s: Sync incomplete, drilling down")
		log.Logger.Info("Negentropy sync incomplete, drilling down")

		msgNeg, err := BuildNegentropyMessage(dto.TypeNegMsg, filter, msg, "", []string{}, []byte{})
		if err != nil {
			log.Logger.Error("Error building Negentropy message",
				zap.Error(err),
			)
			return err
		}
		ws.ChanSender <- msgNeg
	}
	return nil
}
func BuildNegentropyMessage(msgType string, filter nostr.Filter, msgBytes []byte, errMsg string, needIds []string, haveBytes []byte) ([]string, error) {
	var msgArray []string
	msgArray = append(msgArray, msgType)
	msgArray = append(msgArray, "N")
	msgString := hex.EncodeToString(msgBytes)
	switch msgType {
	case "NEG-OPEN":
		jsonFilter, err := json.Marshal(filter)
		if err != nil {
			return nil, err
		}
		msgArray = append(msgArray, string(jsonFilter))
		msgArray = append(msgArray, strconv.Itoa(IdSize))
		msgArray = append(msgArray, msgString)
	case "NEG-MSG":
		msgArray = append(msgArray, msgString)
	case "NEG-ERR":
		msgArray = append(msgArray, errMsg)
	case "NEG-CLOSE":
	case "NEG-HAVE":

		msgArray = append(msgArray, string(haveBytes))
	case "NEG-NEED":
		jsonBytes, err := json.Marshal(needIds)
		if err != nil {
			//logging.Infof("Error marshaling to JSON:%s", err)
			return nil, err
		}
		msgArray = append(msgArray, string(jsonBytes))

	default:
		return nil, errors.New("unknown message type")
	}
	return msgArray, nil
}

func LoadEventVector(events []*nostr.Event) (*negentropy.Vector, error) {
	vector := negentropy.NewVector()
	for _, event := range events {
		id, err := hex.DecodeString(event.ID)
		if err != nil {
			return nil, err
		}

		err = vector.Insert(uint64(event.CreatedAt), id[:IdSize])
		if err != nil {
			return nil, err
		}
	}

	err := vector.Seal()
	if err != nil {
		return nil, err
	}

	return vector, nil
}

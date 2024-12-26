package event

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	nostr_custom "github.com/gabrielmoura/nostr-relay-server/infra/nostr-custom"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
)

func handleJobRequest(ws *dto.WsServer, evt *nostr.Event) string {
	//{
	//"id":"18d788173930d4723b0f11926c1b018ef3132c94c7d1f10db850973bdd1c8a91",
	//"pubkey":"13e6d2aafe13ccebda7c58bb6acc2ba53ae48f6714688663fbd6395642260c44",
	//"created_at":1735252727,
	//"kind":5300,
	//"tags":[
	//["p","99defd55b4d923563129b38dcd17db51023e4e5740d2ece328db10ff45e5b267"],
	//["alt","NIP90 Content Discovery request"],
	//["relays","ws://192.168.1.103:9090/relay"],
	//["param","max_results","200"],
	//["param","user","13e6d2aafe13ccebda7c58bb6acc2ba53ae48f6714688663fbd6395642260c44"]
	//],"content":"","sig":"e074820d4bedaa491b23c668b27db69d787ce00c767c9e5489fd81012c983d6b01defa3a6287076c8ebbccdcc2e8f9ea8eae7fd54991c68858e488d0ab5f3214"}

	resp := nostr.Event{
		Kind:      nostr_custom.KindContentDiscoveryResponse,
		PubKey:    config.Cfg.RelayInformation.PubKey,
		CreatedAt: nostr.Now(),
		Content:   "",
		Tags: nostr.Tags{
			{"e", "13e6d2aafe13ccebda7c58bb6acc2ba53ae48f6714688663fbd6395642260c44"},
		},
	}
	resp.Sign(config.Cfg.RelayInformation.PrivKey)

	ws.ChanSender <- nostr.EventEnvelope{Event: resp}

	return ""
}

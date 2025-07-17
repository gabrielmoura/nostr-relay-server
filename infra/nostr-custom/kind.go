package nostr_custom

const (
	//KindFormattedArticles = 30023
	KindRelay = 411
	//KindBlossom           = 117

	// KindEditContent is the kind of event that represents an edit in the content of a note (EXPERIMENTAL).
	KindEditContent = 1010

	// https://www.data-vending-machines.org/ranges/53xx/ https://github.com/nostr-protocol/nips/blob/vending-machine/90.md
	KindContentDiscovery         = 5300
	KindContentDiscoveryResponse = 6300
	KindPeopleDiscovery          = 5301
	KindPeopleDiscoveryResponse  = 6301
	KindContentSearch            = 5302
	KindContentSearchResponse    = 6302
	KindPeopleSearch             = 5303
	KindPeopleSearchResponse     = 6303
	KindMalwareScanning          = 5500
	KindMalwareScanningResponse  = 6500
	KindEventCount               = 5400
	KindEventCountResponse       = 6400

	// https://github.com/nostr-protocol/nips/blob/master/89.md
	KindRecommendAppPublisher = 31990
	KindRecommendAppReq       = 31989
	KindVanish                = 62
)

func IsJobRequest(kind int) bool {
	return kind > 5000 && kind < 5999
}

func IsJobResponse(kind int) bool {
	return kind > 6000 && kind < 6999
}

func IsJobFeedback(kind int) bool {
	return kind == 7000
}

<!--
Generated from nostr-protocol/nips README.md at commit f9c48b38c0e58119fb9b1cdc38a9179586826414.
Source: https://github.com/nostr-protocol/nips/blob/f9c48b38c0e58119fb9b1cdc38a9179586826414/README.md
-->

## Event Kinds
| kind          | description                     | NIP                                    |
| ------------- | ------------------------------- | -------------------------------------- |
| `0`           | User Metadata                   | [01](01.md)                            |
| `1`           | Short Text Note                 | [10](10.md)                            |
| `2`           | Recommend Relay                 | 01 (deprecated)                        |
| `3`           | Follows                         | [02](02.md)                            |
| `4`           | Encrypted Direct Messages       | [04](04.md)                            |
| `5`           | Event Deletion Request          | [09](09.md)                            |
| `6`           | Repost                          | [18](18.md)                            |
| `7`           | Reaction                        | [25](25.md)                            |
| `8`           | Badge Award                     | [58](58.md)                            |
| `9`           | Chat Message                    | [C7](C7.md)                            |
| `10`          | Group Chat Threaded Reply       | 29 (deprecated)                        |
| `11`          | Thread                          | [7D](7D.md)                            |
| `12`          | Group Thread Reply              | 29 (deprecated)                        |
| `13`          | Seal                            | [59](59.md)                            |
| `14`          | Direct Message                  | [17](17.md)                            |
| `15`          | File Message                    | [17](17.md)                            |
| `16`          | Generic Repost                  | [18](18.md)                            |
| `17`          | Reaction to a website           | [25](25.md)                            |
| `20`          | Picture                         | [68](68.md)                            |
| `21`          | Video Event                     | [71](71.md)                            |
| `22`          | Short-form Portrait Video Event | [71](71.md)                            |
| `24`          | Public Message                  | [A4](A4.md)                            |
| `30`          | internal reference              | [NKBIP-03]                             |
| `31`          | external web reference          | [NKBIP-03]                             |
| `32`          | hardcopy reference              | [NKBIP-03]                             |
| `33`          | prompt reference                | [NKBIP-03]                             |
| `40`          | Channel Creation                | [28](28.md)                            |
| `41`          | Channel Metadata                | [28](28.md)                            |
| `42`          | Channel Message                 | [28](28.md)                            |
| `43`          | Channel Hide Message            | [28](28.md)                            |
| `44`          | Channel Mute User               | [28](28.md)                            |
| `62`          | Request to Vanish               | [62](62.md)                            |
| `64`          | Chess (PGN)                     | [64](64.md)                            |
| `443`         | KeyPackage                      | [Marmot](marmot)                       |
| `444`         | Welcome Message                 | [Marmot](marmot)                       |
| `445`         | Group Event                     | [Marmot](marmot)                       |
| `818`         | Merge Requests                  | [54](54.md)                            |
| `1018`        | Poll Response                   | [88](88.md)                            |
| `1021`        | Bid                             | [15](15.md)                            |
| `1022`        | Bid confirmation                | [15](15.md)                            |
| `1040`        | OpenTimestamps                  | [03](03.md)                            |
| `1059`        | Gift Wrap                       | [59](59.md)                            |
| `1063`        | File Metadata                   | [94](94.md)                            |
| `1068`        | Poll                            | [88](88.md)                            |
| `1111`        | Comment                         | [22](22.md)                            |
| `1222`        | Voice Message                   | [A0](A0.md)                            |
| `1244`        | Voice Message Comment           | [A0](A0.md)                            |
| `1311`        | Live Chat Message               | [53](53.md)                            |
| `1337`        | Code Snippet                    | [C0](C0.md)                            |
| `1617`        | Patches                         | [34](34.md)                            |
| `1618`        | Pull Requests                   | [34](34.md)                            |
| `1619`        | Pull Request Updates            | [34](34.md)                            |
| `1621`        | Issues                          | [34](34.md)                            |
| `1622`        | Git Replies (deprecated)        | [34](34.md)                            |
| `1630`-`1633` | Status                          | [34](34.md)                            |
| `1971`        | Problem Tracker                 | [nostrocket][nostrocket]               |
| `1984`        | Reporting                       | [56](56.md)                            |
| `1985`        | Label                           | [32](32.md)                            |
| `1986`        | Relay reviews                   |                                        |
| `1987`        | AI Embeddings / Vector lists    | [NKBIP-02]                             |
| `2003`        | Torrent                         | [35](35.md)                            |
| `2004`        | Torrent Comment                 | [35](35.md)                            |
| `2022`        | Coinjoin Pool                   | [joinstr][joinstr]                     |
| `4550`        | Community Post Approval         | [72](72.md)                            |
| `5000`-`5999` | Job Request                     | [90](90.md)                            |
| `6000`-`6999` | Job Result                      | [90](90.md)                            |
| `7000`        | Job Feedback                    | [90](90.md)                            |
| `7374`        | Reserved Cashu Wallet Tokens    | [60](60.md)                            |
| `7375`        | Cashu Wallet Tokens             | [60](60.md)                            |
| `7376`        | Cashu Wallet History            | [60](60.md)                            |
| `7516`        | Geocache log                    | [geocaching][geocaching]               |
| `7517`        | Geocache proof of find          | [geocaching][geocaching]               |
| `8000`        | Add User                        | [43](43.md)                            |
| `8001`        | Remove User                     | [43](43.md)                            |
| `9000`-`9030` | Group Control Events            | [29](29.md)                            |
| `9041`        | Zap Goal                        | [75](75.md)                            |
| `9321`        | Nutzap                          | [61](61.md)                            |
| `9467`        | Tidal login                     | [Tidal-nostr]                          |
| `9734`        | Zap Request                     | [57](57.md)                            |
| `9735`        | Zap                             | [57](57.md)                            |
| `9802`        | Highlights                      | [84](84.md)                            |
| `10000`       | Mute list                       | [51](51.md)                            |
| `10001`       | Pin list                        | [51](51.md)                            |
| `10002`       | Relay List Metadata             | [65](65.md), [51](51.md)               |
| `10003`       | Bookmark list                   | [51](51.md)                            |
| `10004`       | Communities list                | [51](51.md)                            |
| `10005`       | Public chats list               | [51](51.md)                            |
| `10006`       | Blocked relays list             | [51](51.md)                            |
| `10007`       | Search relays list              | [51](51.md)                            |
| `10009`       | User groups                     | [51](51.md), [29](29.md)               |
| `10011`       | External Identities             | [39](39.md)                            |
| `10012`       | Favorite relays list            | [51](51.md)                            |
| `10013`       | Private event relay list        | [37](37.md)                            |
| `10015`       | Interests list                  | [51](51.md)                            |
| `10019`       | Nutzap Mint Recommendation      | [61](61.md)                            |
| `10020`       | Media follows                   | [51](51.md)                            |
| `10030`       | User emoji list                 | [51](51.md)                            |
| `10050`       | Relay list to receive DMs       | [51](51.md), [17](17.md)               |
| `10051`       | KeyPackage Relays List          | [Marmot](marmot)                       |
| `10063`       | User server list                | [Blossom][blossom]                     |
| `10096`       | File storage server list        | [96](96.md) (deprecated)               |
| `10166`       | Relay Monitor Announcement      | [66](66.md)                            |
| `10312`       | Room Presence                   | [53](53.md)                            |
| `10377`       | Proxy Announcement              | [Nostr Epoxy][nostr-epoxy]             |
| `11111`       | Transport Method Announcement   | [Nostr Epoxy][nostr-epoxy]             |
| `13194`       | Wallet Info                     | [47](47.md)                            |
| `13534`       | Membership Lists                | [43](43.md)                            |
| `14388`       | User Sound Effect Lists         | [Corny Chat][cornychat-usersoundlist]  |
| `17375`       | Cashu Wallet Event              | [60](60.md)                            |
| `21000`       | Lightning Pub RPC               | [Lightning.Pub][lnpub]                 |
| `22242`       | Client Authentication           | [42](42.md)                            |
| `23194`       | Wallet Request                  | [47](47.md)                            |
| `23195`       | Wallet Response                 | [47](47.md)                            |
| `24133`       | Nostr Connect                   | [46](46.md)                            |
| `24242`       | Blobs stored on mediaservers    | [Blossom][blossom]                     |
| `27235`       | HTTP Auth                       | [98](98.md)                            |
| `28934`       | Join Request                    | [43](43.md)                            |
| `28935`       | Invite Request                  | [43](43.md)                            |
| `28936`       | Leave Request                   | [43](43.md)                            |
| `30000`       | Follow sets                     | [51](51.md)                            |
| `30001`       | Generic lists                   | 51 (deprecated)                        |
| `30002`       | Relay sets                      | [51](51.md)                            |
| `30003`       | Bookmark sets                   | [51](51.md)                            |
| `30004`       | Curation sets                   | [51](51.md)                            |
| `30005`       | Video sets                      | [51](51.md)                            |
| `30006`       | Picture sets                    | [51](51.md)                            |
| `30007`       | Kind mute sets                  | [51](51.md)                            |
| `30008`       | Profile Badges                  | [58](58.md)                            |
| `30009`       | Badge Definition                | [58](58.md)                            |
| `30015`       | Interest sets                   | [51](51.md)                            |
| `30017`       | Create or update a stall        | [15](15.md)                            |
| `30018`       | Create or update a product      | [15](15.md)                            |
| `30019`       | Marketplace UI/UX               | [15](15.md)                            |
| `30020`       | Product sold as an auction      | [15](15.md)                            |
| `30023`       | Long-form Content               | [23](23.md)                            |
| `30024`       | Draft Long-form Content         | [23](23.md)                            |
| `30030`       | Emoji sets                      | [51](51.md)                            |
| `30040`       | Curated Publication Index       | [NKBIP-01]                             |
| `30041`       | Curated Publication Content     | [NKBIP-01]                             |
| `30063`       | Release artifact sets           | [51](51.md)                            |
| `30078`       | Application-specific Data       | [78](78.md)                            |
| `30166`       | Relay Discovery                 | [66](66.md)                            |
| `30267`       | App curation sets               | [51](51.md)                            |
| `30311`       | Live Event                      | [53](53.md)                            |
| `30312`       | Interactive Room                | [53](53.md)                            |
| `30313`       | Conference Event                | [53](53.md)                            |
| `30315`       | User Statuses                   | [38](38.md)                            |
| `30382`       | User Trusted Assertion          | [85](85.md)                            |
| `30383`       | Event Trusted Assertion         | [85](85.md)                            |
| `30384`       | Addressable Trusted Assertion   | [85](85.md)                            |
| `30388`       | Slide Set                       | [Corny Chat][cornychat-slideset]       |
| `30402`       | Classified Listing              | [99](99.md)                            |
| `30403`       | Draft Classified Listing        | [99](99.md)                            |
| `30617`       | Repository announcements        | [34](34.md)                            |
| `30618`       | Repository state announcements  | [34](34.md)                            |
| `30818`       | Wiki article                    | [54](54.md)                            |
| `30819`       | Redirects                       | [54](54.md)                            |
| `31234`       | Draft Event                     | [37](37.md)                            |
| `31388`       | Link Set                        | [Corny Chat][cornychat-linkset]        |
| `31890`       | Feed                            | [NUD: Custom Feeds][NUD: Custom Feeds] |
| `31922`       | Date-Based Calendar Event       | [52](52.md)                            |
| `31923`       | Time-Based Calendar Event       | [52](52.md)                            |
| `31924`       | Calendar                        | [52](52.md)                            |
| `31925`       | Calendar Event RSVP             | [52](52.md)                            |
| `31989`       | Handler recommendation          | [89](89.md)                            |
| `31990`       | Handler information             | [89](89.md)                            |
| `32267`       | Software Application            |                                        |
| `32388`       | User Room Favorites             | [Corny Chat][cornychat-roomfavorites]  |
| `33388`       | High Scores                     | [Corny Chat][cornychat-highscores]     |
| `34235`       | Addressable Video Event         | [71](71.md)                            |
| `34236`       | Addressable Short Video Event   | [71](71.md)                            |
| `34388`       | Sound Effects                   | [Corny Chat][cornychat-soundeffects]   |
| `34550`       | Community Definition            | [72](72.md)                            |
| `38172`       | Cashu Mint Announcement         | [87](87.md)                            |
| `38173`       | Fedimint Announcement           | [87](87.md)                            |
| `37516`       | Geocache listing                | [geocaching](geocaching)               |
| `38383`       | Peer-to-peer Order events       | [69](69.md)                            |
| `39000-9`     | Group metadata events           | [29](29.md)                            |
| `39089`       | Starter packs                   | [51](51.md)                            |
| `39092`       | Media starter packs             | [51](51.md)                            |
| `39701`       | Web bookmarks                   | [B0](B0.md)                            |

[NUD: Custom Feeds]: https://wikifreedia.xyz/cip-01/
[nostrocket]: https://github.com/nostrocket/NIPS/blob/main/Problems.md
[lnpub]: https://github.com/shocknet/Lightning.Pub/blob/master/proto/autogenerated/client.md
[cornychat-usersoundlist]: https://cornychat.com/datatypes#kind14388usersoundeffectslist
[cornychat-slideset]: https://cornychat.com/datatypes#kind30388slideset
[cornychat-linkset]: https://cornychat.com/datatypes#kind31388linkset
[cornychat-roomfavorites]: https://cornychat.com/datatypes#kind32388roomfavorites
[cornychat-highscores]: https://cornychat.com/datatypes#kind33388highscores
[cornychat-soundeffects]: https://cornychat.com/datatypes#kind34388soundeffectsets
[joinstr]: https://gitlab.com/1440000bytes/joinstr/-/blob/main/NIP.md
[NKBIP-01]: https://wikistr.com/nkbip-01*fd208ee8c8f283780a9552896e4823cc9dc6bfd442063889577106940fd927c1
[NKBIP-02]: https://wikistr.com/nkbip-02*fd208ee8c8f283780a9552896e4823cc9dc6bfd442063889577106940fd927c1
[NKBIP-03]: https://wikistr.com/nkbip-03*fd208ee8c8f283780a9552896e4823cc9dc6bfd442063889577106940fd927c1
[blossom]: https://github.com/hzrd149/blossom
[Tidal-nostr]: https://wikistr.com/tidal-nostr
[geocaching]: https://nostrhub.io/naddr1qvzqqqrcvypzppscgyy746fhmrt0nq955z6xmf80pkvrat0yq0hpknqtd00z8z68qqgkwet0vdskx6rfdenj6etkv4h8guc6gs5y5
[nostr-epoxy]: https://github.com/Origami74/nostr-epoxy-reverse-proxy
[marmot]: https://github.com/marmot-protocol/marmot


## Message types


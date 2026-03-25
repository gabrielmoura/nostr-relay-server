import type { BannedUser, ConnectionRecord, EventRecord, UserProfile } from "@/types/admin"
import { buildNpubLike } from "@/lib/utils"

const makeUser = (user: Omit<UserProfile, "npub">): UserProfile => ({
  ...user,
  npub: buildNpubLike(user.pubkey),
})

export const mockUsers: UserProfile[] = [
  makeUser({
    pubkey: "satoshi_node_9k2a7f2c9c0",
    displayName: "Satoshi Node",
    handle: "@satoshi_node",
    picture: "https://images.unsplash.com/photo-1637913754840-3c8755479fdd?auto=format&fit=crop&w=160&q=80",
    trustScore: 0.93,
    relayCount: 4,
    metadata: "Operador tecnico ativo no relay",
    status: "online",
  }),
  makeUser({
    pubkey: "relayops_br_1m9q3e2f11",
    displayName: "Relay Ops BR",
    handle: "@relayops_br",
    picture: "https://images.unsplash.com/photo-1721956514577-f6c15d73e585?auto=format&fit=crop&w=160&q=80",
    nip05: "relayops@nostr.br",
    trustScore: 0.89,
    relayCount: 2,
    metadata: "Equipe de operacao com NIP-05 valido",
    status: "online",
  }),
  makeUser({
    pubkey: "noisynet_2v7w91ca45",
    displayName: "Noisy Net",
    handle: "@noisynet",
    picture: "https://images.unsplash.com/photo-1572313019067-0cff58bab68e?auto=format&fit=crop&w=160&q=80",
    trustScore: 0.61,
    relayCount: 1,
    metadata: "Cliente com comportamento intermitente",
    status: "monitor",
  }),
  makeUser({
    pubkey: "lucasdev_7f334a211e",
    displayName: "Lucas Pereira",
    handle: "@lucasdev",
    picture: "https://images.unsplash.com/photo-1637913754840-3c8755479fdd?auto=format&fit=crop&w=160&q=80",
    trustScore: 0.93,
    relayCount: 3,
    metadata: "Perfil com bom score de confianca",
    status: "monitor",
  }),
  makeUser({
    pubkey: "ana_relay_ff1138d12b",
    displayName: "Ana Clara",
    handle: "@ana_relay",
    picture: "https://images.unsplash.com/photo-1721956514577-f6c15d73e585?auto=format&fit=crop&w=160&q=80",
    nip05: "ana@relay.dev",
    trustScore: 0.86,
    relayCount: 4,
    metadata: "Moderadora com NIP-05 valido",
    status: "monitor",
  }),
  makeUser({
    pubkey: "nostrnode_91bc3effd2",
    displayName: "Nostr Node BR",
    handle: "@nostrnode",
    picture: "https://images.unsplash.com/photo-1572313019067-0cff58bab68e?auto=format&fit=crop&w=160&q=80",
    followers: 8400,
    metadata: "Alto alcance; acompanhar reputacao",
    status: "suspect",
  }),
  makeUser({
    pubkey: "bad_actor_7k2m8dd02c",
    displayName: "Bad Actor",
    handle: "@bad_actor",
    trustScore: 0.18,
    metadata: "Flood recorrente",
    status: "banned",
  }),
  makeUser({
    pubkey: "bot_spam_0r9p129fdc",
    displayName: "Bot Spam",
    handle: "@bot_spam",
    trustScore: 0.09,
    metadata: "Phishing automatizado",
    status: "banned",
  }),
]

const satoshiNode = mockUsers[0]!
const relayOps = mockUsers[1]!
const noisyNet = mockUsers[2]!
const badActor = mockUsers[6]!
const botSpam = mockUsers[7]!

export const mockConnections: ConnectionRecord[] = [
  { ws_id: "conn-8f1a", ip: "200.188.10.11", authed: satoshiNode.pubkey, subscription_count: 4 },
  { ws_id: "conn-2be9", ip: "186.201.44.72", authed: "", subscription_count: 1 },
  { ws_id: "conn-a1d2", ip: "177.12.98.31", authed: relayOps.pubkey, subscription_count: 2 },
  { ws_id: "conn-c9f0", ip: "179.55.21.90", authed: noisyNet.pubkey, subscription_count: 1 },
  { ws_id: "conn-e2c7", ip: "201.4.122.5", authed: relayOps.pubkey, subscription_count: 1 },
]

export const seedBannedUsers: BannedUser[] = [
  {
    ...badActor,
    reason: "flood",
    source: "manual",
    bannedAt: "2026-03-20T11:00:00Z",
    durationLabel: "30 dias",
    related_ids: ["6f9c-d12a"],
  },
  {
    ...botSpam,
    reason: "phishing",
    source: "rule",
    bannedAt: "2026-03-22T08:30:00Z",
    durationLabel: "permanente",
    related_ids: ["1aa3-889f"],
  },
]

export const mockEvents: EventRecord[] = [
  {
    id: "6f9c0aaad12a",
    pubkey: relayOps.pubkey,
    kind: 1,
    created_at: 1774345260,
    content: "Atualizacao do cluster relay com rotacao de chaves e failover automatico.",
    tags: [
      ["t", "relay"],
      ["t", "infra"],
      ["score", "0.91"],
    ],
  },
  {
    id: "1f2erunbookf7d",
    pubkey: satoshiNode.pubkey,
    kind: 30023,
    created_at: 1774343700,
    content: "Runbook de indexacao incremental para busca hibrida com cache de autores ativos.",
    tags: [
      ["t", "relay"],
      ["t", "ops"],
      ["t", "search"],
      ["score", "0.84"],
    ],
  },
  {
    id: "tagmonitor4410",
    pubkey: noisyNet.pubkey,
    kind: 7,
    created_at: 1774342200,
    content: "Observacao sobre padroes de abuso e necessidade de reduzir janela temporal na consulta.",
    tags: [
      ["t", "ops"],
      ["t", "abuse"],
      ["score", "0.72"],
    ],
  },
]

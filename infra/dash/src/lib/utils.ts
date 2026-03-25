import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function shortenId(value: string, start = 8, end = 4) {
  if (value.length <= start + end + 3) {
    return value
  }

  return `${value.slice(0, start)}...${value.slice(-end)}`
}

export function formatCount(value: number | null | undefined) {
  if (value == null) {
    return "N/D"
  }

  return new Intl.NumberFormat("pt-BR").format(value)
}

export function formatDateTime(value: number | string | Date) {
  const date = value instanceof Date ? value : new Date(typeof value === "number" ? value * 1000 : value)

  return new Intl.DateTimeFormat("pt-BR", {
    dateStyle: "short",
    timeStyle: "short",
    timeZone: "UTC",
  }).format(date)
}

export function formatRelativeMinutes(minutes: number | null | undefined) {
  if (minutes == null) {
    return "agora"
  }

  if (minutes <= 1) {
    return "agora"
  }

  return `ha ${minutes}m`
}

export function toTitleCase(value: string | null | undefined) {
  return (value ?? "")
    .split(/[_\s-]+/)
    .filter(Boolean)
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join(" ")
}

export function initials(value: string | null | undefined) {
  return (value ?? "")
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("")
}

export function buildNpubLike(pubkey: string) {
  if (pubkey.startsWith("npub")) {
    return pubkey
  }

  return `npub1${pubkey.slice(0, 20)}`
}

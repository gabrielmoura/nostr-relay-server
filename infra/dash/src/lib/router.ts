import { useNavigate as useReactRouterNavigate } from "@tanstack/react-router"

export function useNavigateFrom() {
  return useReactRouterNavigate({ from: "/events/search" })
}
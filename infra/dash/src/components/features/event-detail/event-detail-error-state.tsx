import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { EmptyPanel, ErrorPanel } from "@/components/shared/state-panels"
import { Button } from "@/components/ui/button"
import { RefreshCcw } from "lucide-react"
import { RelaySearchModal } from "./relay-search-modal"
import { RelayResults } from "./relay-search-modal"
import { useFetchEventFromRelaysMutation } from "@/hooks/use-admin-data"
import { ApiError } from "@/services/admin"

type RelayResult = { relay: string; status: string; error?: string }

function isNotFoundError(error: Error) {
  return error instanceof ApiError && error.status === 404
}

interface EventDetailErrorStateProps {
  eventID: string
  error: Error
  onRetry: () => void
}

export function EventDetailErrorState({ eventID, error, onRetry }: EventDetailErrorStateProps) {
  const { t } = useTranslation()
  const mutation = useFetchEventFromRelaysMutation()
  const [open, setOpen] = useState(false)
  const [relayFeedback, setRelayFeedback] = useState<RelayResult[]>([])

  if (!isNotFoundError(error)) {
    return <ErrorPanel description={error.message || t("eventDetail.loadErrorDescription")} onRetry={onRetry} title={t("eventDetail.loadErrorTitle")} />
  }

  const handleSearch = async (relays: string[]) => {
    try {
      const response = await mutation.mutateAsync({ eventID, relays })
      setRelayFeedback(response.relay_results ?? [])
      toast.success(t("eventDetail.foundInRelay", { relay: response.source_relay }))
      setOpen(false)
      onRetry()
    } catch (mutationError) {
      if (mutationError instanceof ApiError && mutationError.details && typeof mutationError.details === "object") {
        const details = mutationError.details as { relay_results?: RelayResult[] }
        setRelayFeedback(details.relay_results ?? [])
      }
      if (mutationError instanceof Error) {
        toast.error(mutationError.message)
      } else {
        toast.error(t("eventDetail.searchRelayError"))
      }
    }
  }

  return (
    <>
      <EmptyPanel
        action={
          <Button onClick={() => setOpen(true)} title={t("eventDetail.findOnRelaysTitle")} type="button">
            <RefreshCcw className="size-4" />
            {t("eventDetail.findOnRelays")}
          </Button>
        }
        description={t("eventDetail.notFoundDescription")}
        title={t("eventDetail.notFoundTitle")}
      />
      <RelaySearchModal
        description={t("eventDetail.searchModalDescription")}
        onOpenChange={setOpen}
        onSearch={handleSearch}
        open={open}
        title={t("eventDetail.searchModalTitle")}
      />
      {relayFeedback.length > 0 && <RelayResults results={relayFeedback} />}
    </>
  )
}
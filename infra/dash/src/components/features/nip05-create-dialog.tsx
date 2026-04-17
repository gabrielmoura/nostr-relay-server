import { useMemo, useState } from "react"
import { Info, Search } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { useInfiniteUserSearch, useUpsertNIP05Mutation } from "@/hooks/use-admin-data"

export function NIP05CreateDialog() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [searchInput, setSearchInput] = useState("")
  const [submittedSearch, setSubmittedSearch] = useState("")
  const [name, setName] = useState("")
  const [selectedPubkey, setSelectedPubkey] = useState("")

  const searchEnabled = submittedSearch.trim().length > 0
  const usersQuery = useInfiniteUserSearch(submittedSearch, { enabled: searchEnabled })
  const upsertMutation = useUpsertNIP05Mutation()

  const users = useMemo(() => {
    if (!searchEnabled) {
      return []
    }
    return (usersQuery.data?.pages.flatMap((page) => page.items) ?? []).slice(0, 12)
  }, [searchEnabled, usersQuery.data?.pages])

  const reset = () => {
    setSearchInput("")
    setSubmittedSearch("")
    setName("")
    setSelectedPubkey("")
  }

  const handleSearch = () => {
    const value = searchInput.trim()
    if (!value) {
      toast.error(t("nip05.create.toastSearchRequired"))
      return
    }
    setSubmittedSearch(value)
  }

  const handleSelectUser = (pubkey: string, fallbackName?: string) => {
    setSelectedPubkey(pubkey)
    if (!name.trim() && fallbackName) {
      setName(fallbackName.trim().toLowerCase())
    }
  }

  const handleSave = async () => {
    const normalizedName = name.trim().toLowerCase()
    if (!normalizedName || !selectedPubkey) {
      toast.error(t("nip05.create.toastUserNameRequired"))
      return
    }

    try {
      await upsertMutation.mutateAsync({ name: normalizedName, pubkey: selectedPubkey })
      toast.success(t("nip05.create.toastSaveSuccess"))
      setOpen(false)
      reset()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("nip05.create.toastSaveFailed"))
    }
  }

  return (
    <Dialog
      onOpenChange={(next) => {
        setOpen(next)
        if (!next) {
          reset()
        }
      }}
      open={open}
    >
      <DialogTrigger asChild>
        <Button>{t("nip05.newAssociation")}</Button>
      </DialogTrigger>
      <DialogContent className="max-h-[85vh] max-w-2xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t("nip05.create.title")}</DialogTitle>
          <DialogDescription>{t("nip05.create.description")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_auto]">
            <Input
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder={t("nip05.create.searchPlaceholder")}
              value={searchInput}
            />
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button onClick={handleSearch} type="button" variant="outline">
                    <Search className="size-4" />
                    {t("common.search")}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("nip05.create.searchTooltip")}</TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>

          {!searchEnabled ? (
            <div className="rounded-[calc(var(--radius)-0.25rem)] border border-dashed border-border bg-muted/20 p-3 text-xs text-muted-foreground">
              <p className="font-medium text-foreground">{t("nip05.create.noAutoListTitle")}</p>
              <p className="mt-1">{t("nip05.create.noAutoListDescription")}</p>
            </div>
          ) : null}

          {searchEnabled && usersQuery.isLoading ? <p className="text-xs text-muted-foreground">{t("nip05.create.loadingUsers")}</p> : null}
          {searchEnabled && usersQuery.isError ? <p className="text-xs text-destructive">{t("nip05.create.searchFailed")}</p> : null}
          {searchEnabled && !usersQuery.isLoading && !usersQuery.isError && users.length === 0 ? (
            <p className="text-xs text-muted-foreground">{t("nip05.create.noUsersFound")}</p>
          ) : null}

          {searchEnabled && users.length > 0 ? (
            <div className="space-y-2">
              <p className="text-xs text-muted-foreground">
                {users.length > 1
                  ? t("nip05.create.multipleFound")
                  : t("nip05.create.singleFound")}
              </p>
              <div className="max-h-56 space-y-2 overflow-auto rounded-[calc(var(--radius)-0.25rem)] border border-border p-2">
                {users.map((user) => {
                  const fallback = user.nip05?.split("@")[0] ?? user.handle?.replace(/^@/, "") ?? ""
                  return (
                    <button
                      className={`w-full min-w-0 rounded-[calc(var(--radius)-0.25rem)] border px-3 py-2 text-left text-sm transition-colors ${
                        selectedPubkey === user.pubkey ? "border-primary bg-primary/10" : "border-border hover:bg-muted"
                      }`}
                      key={user.pubkey}
                      onClick={() => handleSelectUser(user.pubkey, fallback)}
                      type="button"
                    >
                      <p className="font-medium">{user.displayName}</p>
                      <p className="break-all text-xs text-muted-foreground">{user.pubkey}</p>
                    </button>
                  )
                })}
              </div>
            </div>
          ) : null}

          <div className="space-y-1">
            <div className="flex items-center gap-1 text-xs text-muted-foreground">
              <span>{t("nip05.create.nameLabel")}</span>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button className="inline-flex" type="button">
                      <Info className="size-3.5" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent>
                    {t("nip05.create.nameTooltip")}
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
            <Input onChange={(event) => setName(event.target.value)} placeholder={t("nip05.create.namePlaceholder")} value={name} />
          </div>
        </div>

        <DialogFooter>
          <Button onClick={() => setOpen(false)} type="button" variant="outline">
            {t("common.cancel")}
          </Button>
          <Button disabled={upsertMutation.isPending} onClick={() => void handleSave()} type="button">
            {upsertMutation.isPending ? t("common.saving") : t("common.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { PenSquare } from "lucide-react"

import { ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { useNIP86RelayMetadata, useUpdateNIP86RelayMetadataMutation } from "@/hooks/use-admin-data"

export function RelayMetadataForm() {
  const { t } = useTranslation()
  const metadataQuery = useNIP86RelayMetadata()
  const updateMutation = useUpdateNIP86RelayMetadataMutation()
  const [name, setName] = useState("")
  const [description, setDescription] = useState("")

  useEffect(() => {
    if (!metadataQuery.data) {
      return
    }
    setName(metadataQuery.data.name ?? "")
    setDescription(metadataQuery.data.description ?? "")
  }, [metadataQuery.data])

  return (
    <Card className="overflow-hidden border-secondary/80 bg-card/95">
      <CardHeader>
        <CardTitle>{t("nip86.metadata.title")}</CardTitle>
        <CardDescription>{t("nip86.metadata.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {metadataQuery.isLoading ? <LoadingPanel label={t("nip86.metadata.loading")} /> : null}
        {metadataQuery.isError ? <ErrorPanel description={t("nip86.metadata.errorDescription")} onRetry={() => void metadataQuery.refetch()} title={t("nip86.metadata.errorTitle")} /> : null}

        {!metadataQuery.isLoading && !metadataQuery.isError ? (
          <form
            className="space-y-4"
            onSubmit={(event) => {
              event.preventDefault()
              updateMutation.mutate({ name, description })
            }}
          >
            <div className="space-y-2">
              <Label htmlFor="nip86-name">{t("common.name")}</Label>
              <Input id="nip86-name" onChange={(event) => setName(event.target.value)} value={name} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="nip86-description">{t("nip86.metadata.descriptionLabel")}</Label>
              <Textarea id="nip86-description" onChange={(event) => setDescription(event.target.value)} rows={5} value={description} />
            </div>
            <div className="flex justify-end">
              <Button disabled={updateMutation.isPending} type="submit">
                <PenSquare className="size-4" />
                {updateMutation.isPending ? t("common.saving") : t("nip86.metadata.saveAction")}
              </Button>
            </div>
          </form>
        ) : null}
      </CardContent>
    </Card>
  )
}

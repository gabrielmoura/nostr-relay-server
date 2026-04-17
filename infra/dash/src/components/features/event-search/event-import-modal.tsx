import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"

interface ImportFileResult {
  filename: string
  total: number
  inserted: number
  duplicates: number
  invalid: number
  error?: string
}

interface EventImportModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onImport: (files: File[]) => Promise<void>
  importResult: ImportFileResult[]
  isPending: boolean
}

export function EventImportModal({ open, onOpenChange, onImport, importResult, isPending }: EventImportModalProps) {
  const { t } = useTranslation()
  const [selectedFiles, setSelectedFiles] = useState<File[]>([])

  const handleImport = async () => {
    if (selectedFiles.length === 0) return
    try {
      await onImport(selectedFiles)
      toast.success(t("eventSearch.importSuccess"))
    } catch (error) {
      if (error instanceof Error) {
        toast.error(error.message)
      }
    }
  }

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSelectedFiles(Array.from(event.target.files ?? []))
  }

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{t("eventSearch.importTitle")}</DialogTitle>
          <DialogDescription>{t("eventSearch.importDescription")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <Input multiple onChange={handleFileChange} type="file" />

          {selectedFiles.length > 0 && (
            <div className="rounded-md border border-border p-3">
              <p className="mb-2 text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                {t("eventSearch.selectedFiles")}
              </p>
              <div className="space-y-1 text-sm">
                {selectedFiles.map((file) => (
                  <p key={file.name}>{file.name}</p>
                ))}
              </div>
            </div>
          )}

          <div className="flex justify-end">
            <Button disabled={isPending || selectedFiles.length === 0} onClick={handleImport} type="button">
              {isPending ? t("eventSearch.importing") : t("eventSearch.importFiles")}
            </Button>
          </div>

          {importResult.length > 0 && (
            <div className="rounded-md border border-border p-3">
              <p className="mb-2 text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                {t("eventSearch.resultPerFile")}
              </p>
              <div className="space-y-2 text-sm">
                {importResult.map((file) => (
                  <div className="rounded border border-border px-3 py-2" key={file.filename}>
                    <p className="font-medium text-foreground">{file.filename}</p>
                    <p className="text-xs text-muted-foreground">
                      {t("eventSearch.importStats", {
                        total: file.total,
                        inserted: file.inserted,
                        duplicates: file.duplicates,
                        invalid: file.invalid,
                      })}
                    </p>
                    {file.error ? <p className="mt-1 text-xs text-destructive">{t("eventSearch.error")}: {file.error}</p> : null}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
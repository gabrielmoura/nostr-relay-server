import { DMImportPanel } from "@/features/dm-import/dm-import-panel"
import { PageHeader } from "@/components/shared/page-header"
import { useNIP86RelayMetadata } from "@/hooks/use-admin-data"

export function ImportDMPage() {
  const metadata = useNIP86RelayMetadata()
  return <div className="space-y-6"><PageHeader title="Importar DMs" description="Copie mensagens privadas cifradas para este relay sem revelar sua chave ao servidor." /><DMImportPanel mode="admin" localRelayURL={metadata.data?.relay_url} /></div>
}

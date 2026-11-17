import { useEffect, useMemo, useState } from "react";
import { api, errorMessage } from "../api";
import { DataTable, type Column } from "../components/DataTable";
import { Alert, Badge, Metric, PageFrame } from "../components/ui";
import type { PluginCatalogRow } from "../types";

function PluginStatus({ status }: { status: PluginCatalogRow["status"] }) {
  if (status === "approved") return <Badge tone="good">{"Approved"}</Badge>;
  if (status === "disabled") return <Badge tone="warn">{"Disabled"}</Badge>;
  if (status === "rejected") return <Badge tone="danger">{"Rejected"}</Badge>;
  return <Badge tone="neutral">{"Pending review"}</Badge>;
}

export function PluginsPage() {
  const [rows, setRows] = useState<PluginCatalogRow[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    setBusy(true);
    setError("");
    try {
      setRows((await api.plugins()).rows ?? []);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  }

  useEffect(() => { void load(); }, []);

  const counts = useMemo(() => ({
    total: rows.length,
    pending: rows.filter((row) => row.status === "pending").length,
    approved: rows.filter((row) => row.status === "approved").length
  }), [rows]);

  const columns: Column<PluginCatalogRow>[] = [
    { key: "name", label: "Plugin", sortValue: (row) => row.name.toLowerCase(), render: (row) => <div><strong>{row.name}</strong><div className="muted mono">{row.id}</div></div> },
    { key: "version", label: "Version", sortValue: (row) => row.version, render: (row) => <span className="mono">{row.version}</span> },
    { key: "target", label: "Target", render: (row) => <Badge>{row.target === "android" ? "Android" : "Server"}</Badge> },
    { key: "publisher", label: "Publisher", sortValue: (row) => row.author.toLowerCase(), render: (row) => <div>{row.author}<div className="muted mono">{row.signer_key_id}</div></div> },
    { key: "permissions", label: "Permissions", render: (row) => <span>{row.permissions?.length ? row.permissions.join(", ") : "None"}</span> },
    { key: "status", label: "Review", render: (row) => <PluginStatus status={row.status} /> }
  ];

  return (
    <PageFrame
      title={"Plugins"}
      eyebrow={"Signed extension packages · publisher trust · review queue"}
      actions={<button className="btn" type="button" onClick={() => void load()} disabled={busy}>{"Refresh"}</button>}
    >
      {error && <Alert>{error}</Alert>}
      <div className="metric-row">
        <Metric label={"Packages"} value={String(counts.total)} />
        <Metric label={"Pending review"} value={String(counts.pending)} tone={counts.pending > 0 ? "warn" : undefined} />
        <Metric label={"Approved"} value={String(counts.approved)} tone="good" />
      </div>
      <Alert>
        {"Catalog foundation is active. Packages are stored as inert signed archives; approval does not execute code. Runtime loading stays disabled until the sandbox and capability grants are implemented."}
      </Alert>
      <DataTable
        columns={columns}
        rows={rows}
        rowKey={(row) => row.id + "@" + row.version}
        loading={busy && rows.length === 0}
        empty={"No plugin packages have been submitted."}
      />
    </PageFrame>
  );
}

import { useEffect, useMemo, useState } from "react";
import { api, errorMessage } from "../api";
import { DataTable, type Column } from "../components/DataTable";
import { Alert, Badge, Metric, PageFrame } from "../components/ui";
import type { PluginCatalogEventRow, PluginCatalogRow, PluginPublisherRow } from "../types";

function PluginStatus({ status }: { status: PluginCatalogRow["status"] }) {
  if (status === "approved") return <Badge tone="good">{"Approved"}</Badge>;
  if (status === "disabled") return <Badge tone="warn">{"Disabled"}</Badge>;
  if (status === "rejected") return <Badge tone="danger">{"Rejected"}</Badge>;
  return <Badge tone="neutral">{"Pending review"}</Badge>;
}

export function PluginsPage() {
  const [rows, setRows] = useState<PluginCatalogRow[]>([]);
  const [publishers, setPublishers] = useState<PluginPublisherRow[]>([]);
  const [events, setEvents] = useState<PluginCatalogEventRow[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    setBusy(true);
    setError("");
    try {
      const response = await api.plugins();
      setRows(response.rows ?? []);
      setPublishers(response.publishers ?? []);
      setEvents(response.events ?? []);
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
    approved: rows.filter((row) => row.status === "approved").length,
    disabled: rows.filter((row) => row.status === "disabled").length,
    rejected: rows.filter((row) => row.status === "rejected").length
  }), [rows]);

  const columns: Column<PluginCatalogRow>[] = [
    { key: "name", label: "Plugin", sortValue: (row) => row.name.toLowerCase(), render: (row) => <div><strong>{row.name}</strong><div className="muted mono">{row.id}</div><div className="muted">{row.description}</div></div> },
    { key: "version", label: "Version", sortValue: (row) => row.version, render: (row) => <span className="mono">{row.version}</span> },
    { key: "target", label: "Target", render: (row) => <Badge>{row.target === "android" ? "Android" : "Server"}</Badge> },
    { key: "publisher", label: "Publisher", sortValue: (row) => row.author.toLowerCase(), render: (row) => <div>{row.author}<div className="muted mono">{row.signer_key_id}</div></div> },
    { key: "entrypoint", label: "Entrypoint", render: (row) => <span className="mono">{row.entrypoint}</span> },
    { key: "permissions", label: "Permissions", render: (row) => <span>{row.permissions?.length ? row.permissions.join(", ") : "None"}</span> },
    { key: "digest", label: "Package SHA-256", render: (row) => <span className="mono" title={row.sha256}>{row.sha256.slice(0, 16)}…</span> },
    { key: "submitted", label: "Submitted by", render: (row) => <span>{row.submitted_by || "—"}</span> },
    { key: "review", label: "Review", render: (row) => <div><PluginStatus status={row.status} /><div className="muted">{row.reviewed_by || "Not reviewed"}</div>{row.review_note && <div>{row.review_note}</div>}</div> },
    { key: "updated", label: "Updated", sortValue: (row) => row.updated_at, render: (row) => <span>{new Date(row.updated_at).toLocaleString()}</span> }
  ];


  const publisherColumns: Column<PluginPublisherRow>[] = [
    { key: "label", label: "Publisher", sortValue: (row) => row.label.toLowerCase(), render: (row) => <div><strong>{row.label}</strong><div className="muted mono">{row.key_id}</div></div> },
    { key: "status", label: "Trust", render: (row) => <Badge tone={row.status === "active" ? "good" : "danger"}>{row.status === "active" ? "Trusted" : "Revoked"}</Badge> },
    { key: "created", label: "Added by", render: (row) => <div>{row.created_by}<div className="muted">{new Date(row.created_at).toLocaleString()}</div></div> },
    { key: "revoked", label: "Revoked by", render: (row) => row.revoked_by || "—" }
  ];

  const eventColumns: Column<PluginCatalogEventRow>[] = [
    { key: "plugin", label: "Package", render: (row) => <div><strong>{row.plugin_id}</strong><div className="muted mono">{row.version}</div></div> },
    { key: "action", label: "Event", render: (row) => <Badge>{row.action}</Badge> },
    { key: "actor", label: "Actor", render: (row) => row.actor },
    { key: "note", label: "Review note", render: (row) => row.note || "—" },
    { key: "created", label: "Date", sortValue: (row) => row.created_at, render: (row) => new Date(row.created_at).toLocaleString() }
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
        <Metric label={"Disabled"} value={String(counts.disabled)} />
        <Metric label={"Rejected"} value={String(counts.rejected)} />
      </div>
      <Alert>
        {"Read-only catalog view: package metadata, digest, submitter, and review notes are visible. Upload, publisher-key management, and approve/reject actions are not exposed yet. Packages remain inert; approval does not execute code, and runtime loading stays disabled until the sandbox and capability broker are implemented."}
      </Alert>
      <DataTable
        columns={columns}
        rows={rows}
        rowKey={(row) => row.id + "@" + row.version}
        loading={busy && rows.length === 0}
        empty={"No plugin packages have been submitted."}
      />
      <section className="panel-section">
        <h2>{"Publisher trust roots"}</h2>
        <p className="muted">{"Public signing-key registry. Private signing keys are never stored or displayed here."}</p>
        <DataTable columns={publisherColumns} rows={publishers} rowKey={(row) => row.key_id} empty={"No publisher keys registered."} />
      </section>
      <section className="panel-section">
        <h2>{"Review history"}</h2>
        <DataTable columns={eventColumns} rows={events} rowKey={(row) => String(row.id)} empty={"No review events recorded."} />
      </section>
    </PageFrame>
  );
}

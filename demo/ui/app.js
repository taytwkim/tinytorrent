const peerData = [
  {
    name: "Peer A",
    role: "Seeder",
    status: "Online",
    socket: "/tmp/tinytorrentA.sock",
    exportDir: "./peerA_export",
    accent: "var(--peer-a)",
    stats: { files: 2, pieces: 10, transfers: 0 },
  },
  {
    name: "Peer B",
    role: "Bridge",
    status: "Online",
    socket: "/tmp/tinytorrentB.sock",
    exportDir: "./peerB_export",
    accent: "var(--peer-b)",
    stats: { files: 2, pieces: 10, transfers: 1 },
  },
  {
    name: "Peer C",
    role: "Leecher",
    status: "Online",
    socket: "/tmp/tinytorrentC.sock",
    exportDir: "./peerC_export",
    accent: "var(--peer-c)",
    stats: { files: 0, pieces: 4, transfers: 1 },
  },
];

const files = [
  {
    filename: "foo.txt",
    manifestCid: "bafy-demo-foo-manifest",
    size: "18 B",
    pieces: 1,
    providers: ["Peer A", "Peer B"],
    targetPeer: "Peer B",
    selected: false,
  },
  {
    filename: "pieces.txt",
    manifestCid: "bafy-demo-pieces-manifest",
    size: "15 B",
    pieces: 4,
    providers: ["Peer A", "Peer B"],
    targetPeer: "Peer B",
    selected: true,
  },
];

const transfer = {
  target: "Peer B",
  source: "Peer A",
  manifestCid: "bafy-demo-pieces-manifest",
  file: "pieces.txt",
  completed: 3,
  total: 4,
  activePeers: 2,
  throughput: "1.8 MiB/s",
  pieces: [
    { index: 0, state: "done", owner: "Peer A" },
    { index: 1, state: "done", owner: "Peer B" },
    { index: 2, state: "active", owner: "Peer A" },
    { index: 3, state: "idle", owner: "Pending" },
  ],
};

function renderStatusStrip() {
  const root = document.getElementById("status-strip");
  const cards = [
    { value: "3", label: "Peers online" },
    { value: "2", label: "Seeded files" },
  ];

  root.innerHTML = cards
    .map(
      ({ value, label }) => `
        <div class="status-card">
          <strong>${value}</strong>
          <span>${label}</span>
        </div>
      `,
    )
    .join("");
}

function renderPeers() {
  const root = document.getElementById("peer-grid");
  root.innerHTML = peerData
    .map(
      (peer) => `
        <article class="peer-card" style="--peer-color: ${peer.accent}">
          <div class="peer-card__top">
            <div>
              <h3>${peer.name}</h3>
            </div>
            <span class="chip chip--online">${peer.status}</span>
          </div>

          <div class="peer-card__meta">
            <div class="meta-row">
              <span>RPC socket</span>
              <span>${peer.socket}</span>
            </div>
            <div class="meta-row">
              <span>Export dir</span>
              <span>${peer.exportDir}</span>
            </div>
          </div>

          <div class="peer-stats">
            <div class="stat">
              <strong>${peer.stats.files}</strong>
              <span>files</span>
            </div>
            <div class="stat">
              <strong>${peer.stats.pieces}</strong>
              <span>pieces</span>
            </div>
            <div class="stat">
              <strong>${peer.stats.transfers}</strong>
              <span>active</span>
            </div>
          </div>

          <div class="peer-card__actions">
            <button class="button button--ghost button--small" type="button">View Files</button>
            <button class="button button--ghost button--small" type="button">Tail Log</button>
          </div>
        </article>
      `,
    )
    .join("");
}

function renderFiles() {
  const root = document.getElementById("file-list");
  root.innerHTML = files
    .map(
      (file) => `
        <article class="file-card ${file.selected ? "file-card--selected" : ""}">
          <div class="file-card__top">
            <div>
              <h4>${file.filename}</h4>
              <div class="role">${file.manifestCid}</div>
            </div>
            <div class="file-card__actions">
              <div class="field">
                <label>Download to</label>
                <select class="select" aria-label="Download ${file.filename} to peer">
                  ${peerData
                    .map(
                      (peer) => `
                        <option ${peer.name === file.targetPeer ? "selected" : ""}>${peer.name}</option>
                      `,
                    )
                    .join("")}
                </select>
              </div>
              <button class="button button--primary button--small" type="button">Fetch</button>
            </div>
          </div>

          <div class="file-card__meta">
            <span>${file.size}</span>
            <span>${file.pieces} pieces</span>
            <span>Providers: ${file.providers.join(", ")}</span>
          </div>
        </article>
      `,
    )
    .join("");
}

function renderTransferSummary() {
  const root = document.getElementById("transfer-summary");
  const summary = [
    { value: transfer.file, label: "selected file" },
    { value: transfer.target, label: "downloader" },
    { value: `${transfer.completed}/${transfer.total}`, label: "pieces retrieved" },
    { value: transfer.activePeers, label: "providers engaged" },
    { value: transfer.throughput, label: "current throughput" },
  ];

  root.innerHTML = summary
    .map(
      (item) => `
        <div class="summary-card">
          <strong>${item.value}</strong>
          <span>${item.label}</span>
        </div>
      `,
    )
    .join("");
}

function renderPieces() {
  const root = document.getElementById("piece-grid");
  root.innerHTML = transfer.pieces
    .map(
      (piece) => `
        <div class="piece piece--${piece.state}">
          <strong>Piece ${piece.index}</strong>
          <span>${piece.owner}</span>
        </div>
      `,
    )
    .join("");
}

function init() {
  renderStatusStrip();
  renderPeers();
  renderFiles();
  renderTransferSummary();
  renderPieces();
}

init();

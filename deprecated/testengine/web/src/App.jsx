import { useCallback, useEffect, useRef, useState } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  addEdge,
  MarkerType,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import { BLOCKS } from './blocks';
import BlockNode from './BlockNode';

const nodeTypes = { block: BlockNode };

const STATUS_COLORS = {
  '': '#6b7280',
  passed: '#34d399',
  failed: '#f87171',
  blocked: '#fbbf24',
  skipped: '#9ca3af',
};
const STATUSES = ['', 'passed', 'failed', 'blocked', 'skipped'];

function edgeVisual(on) {
  const color = STATUS_COLORS[on] ?? '#6b7280';
  return {
    label: on || 'any',
    style: { stroke: color, strokeWidth: 1.5 },
    labelStyle: { fill: color, fontSize: 11, fontFamily: 'ui-monospace, monospace' },
    labelBgStyle: { fill: '#11141a' },
    markerEnd: { type: MarkerType.ArrowClosed, color },
  };
}

// Convert the React Flow graph into exactly the JSON the engine consumes.
function toEngineGraph(nodes, edges, startId) {
  return {
    start: startId || nodes[0]?.id || '',
    nodes: nodes.map((n) => {
      const out = { id: n.id, type: n.data.blockType };
      if (n.data.config && Object.keys(n.data.config).length) out.config = n.data.config;
      if (n.data.rules && n.data.rules.length) out.rules = n.data.rules;
      return out;
    }),
    edges: edges.map((e) => {
      const out = { from: e.source, to: e.target };
      if (e.data?.on) out.on = e.data.on;
      return out;
    }),
  };
}

export default function App() {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [startId, setStartId] = useState('');
  const [selected, setSelected] = useState(null); // { kind: 'node' | 'edge', id }
  const [showExport, setShowExport] = useState(false);
  const idRef = useRef(1);

  const [cfgText, setCfgText] = useState('');
  const [cfgErr, setCfgErr] = useState('');
  const [rulesText, setRulesText] = useState('');
  const [rulesErr, setRulesErr] = useState('');

  const selectedNode = selected?.kind === 'node' ? nodes.find((n) => n.id === selected.id) : null;
  const selectedEdge = selected?.kind === 'edge' ? edges.find((e) => e.id === selected.id) : null;

  // Keep the "start" flag on nodes in sync with startId (drives the badge).
  useEffect(() => {
    setNodes((nds) => nds.map((n) => ({ ...n, data: { ...n.data, isStart: n.id === startId } })));
  }, [startId, setNodes]);

  // Load editor buffers when the selected node changes.
  useEffect(() => {
    if (selectedNode) {
      setCfgText(JSON.stringify(selectedNode.data.config ?? {}, null, 2));
      setRulesText(JSON.stringify(selectedNode.data.rules ?? [], null, 2));
      setCfgErr('');
      setRulesErr('');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selected?.id]);

  const onConnect = useCallback(
    (params) => setEdges((eds) => addEdge({ ...params, data: { on: '' }, ...edgeVisual('') }, eds)),
    [setEdges],
  );

  const addBlock = (type) => {
    const def = BLOCKS[type];
    const id = `n_${idRef.current++}`;
    const node = {
      id,
      type: 'block',
      position: { x: 140 + Math.random() * 160, y: 80 + Math.random() * 220 },
      data: {
        blockType: type,
        label: def.label,
        config: structuredClone(def.defaultConfig ?? {}),
        rules: structuredClone(def.defaultRules ?? []),
        isStart: false,
      },
    };
    setNodes((nds) => nds.concat(node));
    setStartId((s) => s || id); // first block added becomes the start node
  };

  const updateNodeData = (id, patch) =>
    setNodes((nds) => nds.map((n) => (n.id === id ? { ...n, data: { ...n.data, ...patch } } : n)));

  const onCfgChange = (text) => {
    setCfgText(text);
    try {
      const v = JSON.parse(text);
      setCfgErr('');
      if (selectedNode) updateNodeData(selectedNode.id, { config: v });
    } catch {
      setCfgErr('invalid JSON');
    }
  };

  const onRulesChange = (text) => {
    setRulesText(text);
    try {
      const v = JSON.parse(text);
      if (!Array.isArray(v)) {
        setRulesErr('rules must be an array');
        return;
      }
      setRulesErr('');
      if (selectedNode) updateNodeData(selectedNode.id, { rules: v });
    } catch {
      setRulesErr('invalid JSON');
    }
  };

  const setEdgeStatus = (id, on) =>
    setEdges((eds) =>
      eds.map((e) => (e.id === id ? { ...e, data: { ...e.data, on }, ...edgeVisual(on) } : e)),
    );

  const deleteSelected = () => {
    if (selected?.kind === 'node') {
      setNodes((nds) => nds.filter((n) => n.id !== selected.id));
      setEdges((eds) => eds.filter((e) => e.source !== selected.id && e.target !== selected.id));
    } else if (selected?.kind === 'edge') {
      setEdges((eds) => eds.filter((e) => e.id !== selected.id));
    }
    setSelected(null);
  };

  const graphJSON = JSON.stringify(toEngineGraph(nodes, edges, startId), null, 2);
  const copyJSON = () => navigator.clipboard?.writeText(graphJSON);
  const downloadJSON = () => {
    const blob = new Blob([graphJSON], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'graph.json';
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="app">
      <div className="toolbar">
        <span className="brand">
          testengine<span className="dim"> / canvas</span>
        </span>
        <div className="spacer" />
        <button onClick={() => setShowExport((s) => !s)}>{showExport ? 'Hide JSON' : 'Show JSON'}</button>
        <button onClick={copyJSON}>Copy</button>
        <button className="primary" onClick={downloadJSON}>
          Download graph.json
        </button>
      </div>

      <aside className="palette">
        <div className="palette-title">blocks</div>
        {Object.keys(BLOCKS).map((t) => (
          <button key={t} className="palette-item" onClick={() => addBlock(t)}>
            <span className="dot" style={{ background: BLOCKS[t].color }} />
            {BLOCKS[t].label}
          </button>
        ))}
        <div className="hint">click to add · drag to move · drag handle→handle to connect</div>
      </aside>

      <main className="canvas">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          nodeTypes={nodeTypes}
          colorMode="dark"
          onNodeClick={(_, n) => setSelected({ kind: 'node', id: n.id })}
          onEdgeClick={(_, e) => setSelected({ kind: 'edge', id: e.id })}
          onPaneClick={() => setSelected(null)}
          fitView
        >
          <Background gap={16} />
          <Controls />
          <MiniMap pannable zoomable />
        </ReactFlow>
      </main>

      <aside className="panel">
        {!selected && <div className="empty">Select a block or edge to edit it.</div>}

        {selectedNode && (
          <div className="editor">
            <div className="panel-head">
              <span className="ptype">{selectedNode.data.blockType}</span>
              <button
                className="link"
                onClick={() => setStartId(selectedNode.id)}
                disabled={startId === selectedNode.id}
              >
                {startId === selectedNode.id ? 'start node' : 'set as start'}
              </button>
            </div>

            <label>config</label>
            <textarea
              className={`code${cfgErr ? ' err' : ''}`}
              value={cfgText}
              onChange={(e) => onCfgChange(e.target.value)}
              rows={8}
              spellCheck={false}
            />
            {cfgErr && <div className="errline">{cfgErr}</div>}

            <label>rules</label>
            <textarea
              className={`code${rulesErr ? ' err' : ''}`}
              value={rulesText}
              onChange={(e) => onRulesChange(e.target.value)}
              rows={6}
              spellCheck={false}
            />
            {rulesErr && <div className="errline">{rulesErr}</div>}

            <div className="meta">
              <div>
                <span className="k">reads</span>
                {(BLOCKS[selectedNode.data.blockType].reads || []).join(', ') || '—'}
              </div>
              <div>
                <span className="k">writes</span>
                {(BLOCKS[selectedNode.data.blockType].writes || []).join(', ') || '—'}
              </div>
            </div>

            <button className="danger" onClick={deleteSelected}>
              Delete block
            </button>
          </div>
        )}

        {selectedEdge && (
          <div className="editor">
            <div className="panel-head">
              <span className="ptype">edge</span>
            </div>
            <div className="dim small">
              {selectedEdge.source} → {selectedEdge.target}
            </div>
            <label>fires on status</label>
            <select
              value={selectedEdge.data?.on ?? ''}
              onChange={(e) => setEdgeStatus(selectedEdge.id, e.target.value)}
            >
              {STATUSES.map((s) => (
                <option key={s} value={s}>
                  {s || 'any (default)'}
                </option>
              ))}
            </select>
            <button className="danger" onClick={deleteSelected}>
              Delete edge
            </button>
          </div>
        )}
      </aside>

      {showExport && (
        <div className="export">
          <div className="export-head">
            <span>graph.json</span>
            <button className="link" onClick={() => setShowExport(false)}>
              close
            </button>
          </div>
          <pre>{graphJSON}</pre>
        </div>
      )}
    </div>
  );
}

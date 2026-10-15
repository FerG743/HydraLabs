import { Handle, Position } from '@xyflow/react';
import { BLOCKS } from './blocks';

export default function BlockNode({ data, selected }) {
  const color = BLOCKS[data.blockType]?.color || '#9ca3af';
  return (
    <div
      className={`bnode${selected ? ' sel' : ''}${data.isStart ? ' start' : ''}`}
      style={{ '--c': color }}
    >
      <Handle type="target" position={Position.Top} />
      <div className="bnode-type">{data.blockType}</div>
      <div className="bnode-label">{data.label}</div>
      {data.isStart && <span className="bnode-badge">start</span>}
      <Handle type="source" position={Position.Bottom} />
    </div>
  );
}

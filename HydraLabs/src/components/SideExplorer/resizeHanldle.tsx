import React from 'react';

const ResizeHandle = ({ isResizing, onMouseDown }) => {
  return (
    <div
      className={`w-1 bg-gray-200 hover:bg-blue-400 cursor-col-resize transition-all duration-300 relative group ${
        isResizing ? 'bg-blue-500 animate-pulse-subtle' : ''
      }`}
      onMouseDown={onMouseDown}
    >
      <div className="absolute inset-y-0 -inset-x-1 flex items-center justify-center">
        <div className="w-0.5 h-8 bg-gray-400 rounded opacity-0 group-hover:opacity-100 transition-all duration-300 animate-fade-in" />
      </div>
    </div>
  );
};

export default ResizeHandle;
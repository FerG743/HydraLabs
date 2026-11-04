import React from 'react';
import ActivityPanel from '@/components/ActivityPanel/ActivityPanel';
import ApiSetup from '@/components/ApiSetup';
import SideExplorer from '@/components/SideExplorer';

const APITesting = () => {
  return (
    <div className="h-screen bg-background flex flex-col">
      {/* Header */}
      <div className="flex-shrink-0 bg-card border-b border-border px-4 py-3">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-semibold text-foreground">HydraLabs</h1>
          <div className="flex items-center gap-2">
            <button className="px-3 py-1 text-sm bg-muted text-muted-foreground rounded hover:bg-muted/80">
              Save
            </button>
            <button className="px-3 py-1 text-sm bg-primary text-primary-foreground rounded hover:bg-primary/90">
              Settings
            </button>
          </div>
        </div>
      </div>

      {/* Main Layout */}
      <div className="flex-1 flex overflow-hidden min-h-0">
        {/* Sidebar */}
        <SideExplorer />

        {/* Main Content - Request Builder + Response Viewer */}
        <div className="flex-1 flex flex-col min-h-0">
          {/* Request Builder (Top Half) */}
          <div className="flex-1 min-h-0 overflow-hidden">
            <ApiSetup className="h-full" />
          </div>
        </div>
      </div>

      {/* Activity Panel (Bottom) - Resizable - NO WRAPPER! */}
      <ActivityPanel 
        height={300}
        minHeight={150}
        maxHeight={600}
      />
    </div>
  );
};

export default APITesting;
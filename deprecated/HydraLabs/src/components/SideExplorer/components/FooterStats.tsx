import React from 'react';

const FooterStats = ({ activeSection, testSuites, fileTree }) => {
  const totalFiles = fileTree.reduce((acc, folder) => acc + (folder.children?.length || 0), 0);
  const runningTestSuites = testSuites.filter(s => s.status === 'running').length;

  return (
    // FIXED: Changed from light theme to dark theme
    <div className="p-3 border-t border-border bg-card">
      <div className="text-xs text-muted-foreground">
        {activeSection === 'suites' ? (
          <div className="space-y-1">
            <div className="flex justify-between">
              <span>Total Suites:</span>
              <span className="text-foreground">{testSuites.length}</span>
            </div>
            <div className="flex justify-between">
              <span>Running:</span>
              {/* FIXED: Changed yellow to warning color from your theme */}
              <span className="text-warning">{runningTestSuites}</span>
            </div>
          </div>
        ) : (
          <div>
            {totalFiles} files
          </div>
        )}
      </div>
    </div>
  );
};

export default FooterStats;
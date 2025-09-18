import React from 'react';

const FooterStats = ({ activeSection, testSuites, fileTree }) => {
  const totalFiles = fileTree.reduce((acc, folder) => acc + (folder.children?.length || 0), 0);
  const runningTestSuites = testSuites.filter(s => s.status === 'running').length;

  return (
    <div className="p-3 border-t border-gray-200 bg-gray-50">
      <div className="text-xs text-gray-500">
        {activeSection === 'suites' ? (
          <div className="space-y-1">
            <div className="flex justify-between">
              <span>Total Suites:</span>
              <span>{testSuites.length}</span>
            </div>
            <div className="flex justify-between">
              <span>Running:</span>
              <span className="text-yellow-600">{runningTestSuites}</span>
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
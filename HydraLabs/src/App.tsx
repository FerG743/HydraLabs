import React, { useState, useEffect } from 'react';
import { 
  Play, Save, Copy, Eye, EyeOff, Plus, X, Code, FileText, Settings,
  ChevronDown, ChevronRight, AlertCircle, CheckCircle, Clock, Zap,
  Globe, Lock, Database, Folder, Trash2, Download, Send
} from 'lucide-react';

// Mock data based on your components
const defaultTestSuites = [
  {
    id: 'test-suite-1',
    name: 'Login Tests',
    type: 'functional',
    status: 'passed',
    tests: 12,
    passed: 10,
    failed: 2,
    lastRun: '2 min ago'
  },
  {
    id: 'test-suite-2', 
    name: 'API Integration',
    type: 'integration',
    status: 'running',
    tests: 8,
    passed: 5,
    failed: 0,
    lastRun: 'Running...'
  },
  {
    id: 'test-suite-3',
    name: 'Performance Tests',
    type: 'performance',
    status: 'failed',
    tests: 5,
    passed: 2,
    failed: 3,
    lastRun: '1 hour ago'
  }
];

const defaultFileTree = [
  {
    name: 'Collections',
    type: 'folder',
    children: [
      { name: 'User API.json', type: 'file', size: '2.4 KB' },
      { name: 'Auth Tests.json', type: 'file', size: '1.8 KB' },
      { name: 'Product API.json', type: 'file', size: '3.2 KB' }
    ]
  },
  {
    name: 'Environments',
    type: 'folder', 
    children: [
      { name: 'Development.json', type: 'file', size: '890 B' },
      { name: 'Production.json', type: 'file', size: '1.2 KB' }
    ]
  }
];

const defaultActivities = [
  { id: 1, type: 'request', message: 'GET /api/users - 200 OK', timestamp: new Date(), status: 'success' },
  { id: 2, type: 'test', message: 'Login test suite completed', timestamp: new Date(), status: 'success' },
  { id: 3, type: 'error', message: 'Connection timeout to /api/products', timestamp: new Date(), status: 'error' }
];

// Utility functions
const getStatusIcon = (status) => {
  switch(status) {
    case 'passed': return <CheckCircle className="w-4 h-4 text-green-500" />;
    case 'failed': return <X className="w-4 h-4 text-red-500" />;
    case 'running': return <Clock className="w-4 h-4 text-yellow-500 animate-pulse" />;
    default: return <AlertCircle className="w-4 h-4 text-muted-foreground" />;
  }
};

const getTypeColor = (type) => {
  switch(type) {
    case 'unit': return 'bg-blue-100 text-blue-800';
    case 'integration': return 'bg-purple-100 text-purple-800';
    case 'functional': return 'bg-green-100 text-green-800';
    case 'performance': return 'bg-orange-100 text-orange-800';
    default: return 'bg-muted text-gray-800';
  }
};

// File Tree Component
const FileTreeItem = ({ item, level = 0, expandedFolders, onToggleFolder, onFileSelect }) => {
  const isExpanded = expandedFolders.includes(item.name);
  
  return (
    <div>
      <div 
        className="flex items-center gap-1 px-2 py-1 hover:bg-muted/50 cursor-pointer text-sm group"
        style={{ paddingLeft: `${8 + level * 16}px` }}
        onClick={() => {
          if (item.type === 'folder') {
            onToggleFolder(item.name);
          } else if (onFileSelect) {
            onFileSelect(item);
          }
        }}
      >
        {item.type === 'folder' ? (
          <>
            {isExpanded ? 
              <ChevronDown className="w-3 h-3 text-muted-foreground" /> : 
              <ChevronRight className="w-3 h-3 text-muted-foreground" />
            }
            <Folder className="w-4 h-4 text-primary" />
          </>
        ) : (
          <>
            <div className="w-3 h-3" />
            <FileText className="w-4 h-4 text-muted-foreground" />
          </>
        )}
        <span className="truncate flex-1">{item.name}</span>
        {item.size && (
          <span className="text-xs text-muted-foreground opacity-0 group-hover:opacity-100">
            {item.size}
          </span>
        )}
      </div>
      
      {item.type === 'folder' && isExpanded && item.children && (
        <div>
          {item.children.map((child, idx) => (
            <FileTreeItem 
              key={idx} 
              item={child} 
              level={level + 1}
              expandedFolders={expandedFolders}
              onToggleFolder={onToggleFolder}
              onFileSelect={onFileSelect}
            />
          ))}
        </div>
      )}
    </div>
  );
};

// Side Explorer Component
const SideExplorer = ({ onFileSelect, onTestSuiteSelect }) => {
  const [activeSection, setActiveSection] = useState('collections');
  const [expandedFolders, setExpandedFolders] = useState(['Collections']);

  const toggleFolder = (folderName) => {
    setExpandedFolders(prev => 
      prev.includes(folderName) 
        ? prev.filter(f => f !== folderName)
        : [...prev, folderName]
    );
  };

  return (
    <div className="w-64 bg-card border-r border-border flex flex-col h-full">
      {/* Header */}
      <div className="p-3 border-b border-border">
        <div className="flex gap-1">
          <button 
            onClick={() => setActiveSection('collections')}
            className={`flex-1 px-3 py-2 text-xs font-medium rounded transition-colors ${
              activeSection === 'collections' 
                ? 'bg-primary/20 text-primary' 
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            Collections
          </button>
          <button 
            onClick={() => setActiveSection('suites')}
            className={`flex-1 px-3 py-2 text-xs font-medium rounded transition-colors ${
              activeSection === 'suites' 
                ? 'bg-primary/20 text-primary' 
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            Test Suites
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {activeSection === 'collections' && (
          <div className="p-2">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-muted-foreground px-2">FILES</span>
              <button className="p-1 hover:bg-muted/50 rounded">
                <Plus className="w-3 h-3 text-muted-foreground" />
              </button>
            </div>
            {defaultFileTree.map((item, idx) => (
              <FileTreeItem 
                key={idx} 
                item={item}
                expandedFolders={expandedFolders}
                onToggleFolder={toggleFolder}
                onFileSelect={onFileSelect}
              />
            ))}
          </div>
        )}

        {activeSection === 'suites' && (
          <div className="p-2 space-y-2">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-muted-foreground px-2">TEST SUITES</span>
              <button className="p-1 hover:bg-muted/50 rounded">
                <Plus className="w-3 h-3 text-muted-foreground" />
              </button>
            </div>
            {defaultTestSuites.map((suite) => (
              <div 
                key={suite.id}
                className="p-2 rounded border border-border hover:border-primary/50 cursor-pointer transition-colors"
                onClick={() => onTestSuiteSelect && onTestSuiteSelect(suite)}
              >
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm font-medium text-foreground">{suite.name}</span>
                  {getStatusIcon(suite.status)}
                </div>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span className={`px-2 py-1 rounded text-xs ${getTypeColor(suite.type)}`}>
                    {suite.type}
                  </span>
                  <span>{suite.tests} tests</span>
                </div>
                <div className="text-xs text-muted-foreground mt-1">
                  {suite.lastRun}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

// Request Builder Component
const RequestBuilder = ({ onSendRequest, requestData, setRequestData }) => {
  const [activeTab, setActiveTab] = useState('headers');
  const [expandedSections, setExpandedSections] = useState(['headers']);

  const httpMethods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];
  const methodColors = {
    GET: 'bg-green-100 text-green-700',
    POST: 'bg-blue-100 text-blue-700',
    PUT: 'bg-orange-100 text-orange-700',
    PATCH: 'bg-yellow-100 text-yellow-700',
    DELETE: 'bg-red-100 text-red-700'
  };

  const updateRequestData = (path, value) => {
    setRequestData(prev => {
      const newData = { ...prev };
      const keys = path.split('.');
      let current = newData;
      
      for (let i = 0; i < keys.length - 1; i++) {
        current = current[keys[i]];
      }
      
      current[keys[keys.length - 1]] = value;
      return newData;
    });
  };

  return (
    <div className="bg-card border-b border-border p-4">
      {/* URL Bar */}
      <div className="flex gap-2 mb-4">
        <select 
          className={`px-3 py-2 rounded border text-sm font-medium ${methodColors[requestData.method] || 'bg-muted'}`}
          value={requestData.method}
          onChange={(e) => updateRequestData('method', e.target.value)}
        >
          {httpMethods.map(method => (
            <option key={method} value={method}>{method}</option>
          ))}
        </select>
        <input 
          type="text"
          className="flex-1 px-3 py-2 bg-background border border-border rounded focus:ring-2 focus:ring-ring focus:border-transparent text-foreground"
          placeholder="Enter request URL"
          value={requestData.url}
          onChange={(e) => updateRequestData('url', e.target.value)}
        />
        <button 
          onClick={() => onSendRequest(requestData)}
          className="px-6 py-2 bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors flex items-center gap-2"
        >
          <Send className="w-4 h-4" />
          Send
        </button>
      </div>

      {/* Tabs */}
      <div className="border-b border-border mb-4">
        <nav className="flex space-x-8">
          {['headers', 'body', 'auth'].map(tab => (
            <button 
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`py-2 px-1 border-b-2 font-medium text-sm transition-colors ${
                activeTab === tab 
                  ? 'border-primary text-primary' 
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="min-h-32">
        {activeTab === 'headers' && (
          <div className="space-y-2">
            {requestData.headers.map((header, index) => (
              <div key={index} className="flex items-center gap-2">
                <input
                  type="text"
                  placeholder="Header"
                  value={header.key}
                  onChange={(e) => {
                    const newHeaders = [...requestData.headers];
                    newHeaders[index].key = e.target.value;
                    updateRequestData('headers', newHeaders);
                  }}
                  className="flex-1 px-3 py-2 bg-background border border-border rounded focus:ring-1 focus:ring-ring text-foreground"
                />
                <input
                  type="text"
                  placeholder="Value"
                  value={header.value}
                  onChange={(e) => {
                    const newHeaders = [...requestData.headers];
                    newHeaders[index].value = e.target.value;
                    updateRequestData('headers', newHeaders);
                  }}
                  className="flex-1 px-3 py-2 bg-background border border-border rounded focus:ring-1 focus:ring-ring text-foreground"
                />
                <button 
                  onClick={() => {
                    const newHeaders = requestData.headers.filter((_, i) => i !== index);
                    updateRequestData('headers', newHeaders);
                  }}
                  className="p-2 text-destructive hover:text-destructive/80"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            ))}
            <button 
              onClick={() => {
                const newHeaders = [...requestData.headers, { key: '', value: '' }];
                updateRequestData('headers', newHeaders);
              }}
              className="flex items-center gap-2 px-3 py-2 text-primary hover:text-primary/80 hover:bg-primary/10 rounded"
            >
              <Plus className="w-4 h-4" />
              Add Header
            </button>
          </div>
        )}

        {activeTab === 'body' && (
          <div className="space-y-4">
            <div className="flex gap-4">
              {['none', 'json', 'form-data'].map(type => (
                <label key={type} className="flex items-center gap-2">
                  <input 
                    type="radio" 
                    name="bodyType" 
                    value={type}
                    checked={requestData.body.type === type}
                    onChange={(e) => updateRequestData('body.type', e.target.value)}
                  />
                  <span className="text-sm">{type}</span>
                </label>
              ))}
            </div>
            {requestData.body.type === 'json' && (
              <textarea
                placeholder='{\n  "name": "example",\n  "value": "data"\n}'
                value={requestData.body.raw}
                onChange={(e) => updateRequestData('body.raw', e.target.value)}
                rows={6}
                className="w-full px-3 py-2 bg-background border border-border rounded focus:ring-2 focus:ring-ring font-mono text-sm text-foreground"
              />
            )}
          </div>
        )}

        {activeTab === 'auth' && (
          <div className="space-y-4">
            <div className="flex gap-4">
              {['none', 'bearer', 'basic'].map(type => (
                <label key={type} className="flex items-center gap-2">
                  <input 
                    type="radio" 
                    name="authType" 
                    value={type}
                    checked={requestData.auth.type === type}
                    onChange={(e) => updateRequestData('auth.type', e.target.value)}
                  />
                  <span className="text-sm">{type}</span>
                </label>
              ))}
            </div>
            {requestData.auth.type === 'bearer' && (
              <input
                type="text"
                placeholder="Token"
                value={requestData.auth.bearer}
                onChange={(e) => updateRequestData('auth.bearer', e.target.value)}
                className="w-full px-3 py-2 bg-background border border-border rounded focus:ring-2 focus:ring-ring text-foreground"
              />
            )}
          </div>
        )}
      </div>
    </div>
  );
};

// Response Viewer Component
const ResponseViewer = ({ response, isLoading }) => {
  const [activeTab, setActiveTab] = useState('body');

  if (isLoading) {
    return (
      <div className="p-8 text-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500 mx-auto mb-4"></div>
        <p className="text-muted-foreground">Sending request...</p>
      </div>
    );
  }

  if (!response) {
    return (
      <div className="p-8 text-center text-muted-foreground">
        <div className="w-12 h-12 mx-auto mb-4 text-muted-foreground/50">
          <Globe className="w-full h-full" />
        </div>
        <p>Send a request to see the response</p>
      </div>
    );
  }

  return (
    <div className="bg-card h-full flex flex-col">
      {/* Status Bar */}
      <div className="flex justify-between items-center p-4 bg-muted/30 border-b border-border">
        <div className="flex items-center gap-4">
          <span className={`px-3 py-1 rounded text-sm font-medium ${
            response.status >= 200 && response.status < 300 
              ? 'bg-success/20 text-success' 
              : 'bg-destructive/20 text-destructive'
          }`}>
            {response.status} {response.statusText || 'OK'}
          </span>
          {response.time && <span className="text-sm text-muted-foreground">{response.time}ms</span>}
          {response.size && <span className="text-sm text-muted-foreground">{response.size} bytes</span>}
        </div>
        <button className="p-1 hover:bg-muted/50 rounded">
          <Copy className="w-4 h-4 text-muted-foreground" />
        </button>
      </div>

      {/* Response Tabs */}
      <div className="border-b border-border">
        <nav className="flex px-4">
          {['body', 'headers'].map(tab => (
            <button 
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`py-3 px-4 border-b-2 font-medium text-sm transition-colors ${
                activeTab === tab 
                  ? 'border-primary text-primary' 
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </nav>
      </div>

      {/* Response Content */}
      <div className="flex-1 overflow-auto">
        {activeTab === 'body' && (
          <pre className="p-4 text-sm font-mono text-foreground whitespace-pre-wrap">
            {response.error || JSON.stringify(response.data, null, 2)}
          </pre>
        )}
        {activeTab === 'headers' && (
          <div className="p-4">
            {response.headers && Object.entries(response.headers).map(([key, value]) => (
              <div key={key} className="flex py-1 border-b border-border text-sm">
                <div className="w-1/3 font-medium text-muted-foreground">{key}</div>
                <div className="w-2/3 text-foreground">{value}</div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

// Activity Panel Component
const ActivityPanel = ({ activities, onClear }) => {
  const [activeTab, setActiveTab] = useState('activity');

  return (
    <div className="bg-card border-t border-border h-48">
      {/* Header */}
      <div className="flex items-center justify-between p-3 border-b border-border bg-muted/30">
        <div className="flex gap-4">
          {['activity', 'console'].map(tab => (
            <button 
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`text-sm font-medium ${
                activeTab === tab 
                  ? 'text-primary' 
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </div>
        <div className="flex gap-2">
          <button onClick={onClear} className="p-1 hover:text-destructive">
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-2">
        {activeTab === 'activity' && (
          <div className="space-y-1">
            {activities.map((activity, i) => (
              <div key={activity.id || i} className="flex items-center gap-2 px-2 py-1 text-sm hover:bg-muted/30 rounded">
                <div className={`w-2 h-2 rounded-full ${
                  activity.status === 'success' ? 'bg-success' : 
                  activity.status === 'error' ? 'bg-destructive' : 'bg-warning'
                }`} />
                <span className="text-xs text-muted-foreground">
                  {activity.timestamp.toLocaleTimeString()}
                </span>
                <span className="flex-1 text-foreground">{activity.message}</span>
              </div>
            ))}
          </div>
        )}
        {activeTab === 'console' && (
          <div className="p-4 text-center text-muted-foreground text-sm">
            Console output will appear here
          </div>
        )}
      </div>
    </div>
  );
};

// Main App Component
export default function HydraLabsApp() {
  // State management
  const [requestData, setRequestData] = useState({
    method: 'GET',
    url: 'https://jsonplaceholder.typicode.com/users',
    headers: [{ key: 'Content-Type', value: 'application/json' }],
    body: { type: 'none', raw: '' },
    auth: { type: 'none', bearer: '' }
  });

  const [response, setResponse] = useState(null);
  const [isLoading, setIsLoading] = useState(false);
  const [activities, setActivities] = useState(defaultActivities);

  // Handlers
  const handleSendRequest = async (data) => {
    setIsLoading(true);
    setResponse(null);

    // Add activity
    const newActivity = {
      id: Date.now(),
      type: 'request',
      message: `${data.method} ${data.url}`,
      timestamp: new Date(),
      status: 'loading'
    };
    setActivities(prev => [newActivity, ...prev.slice(0, 9)]);

    try {
      // Convert headers array to object
      const headers = {};
      data.headers.forEach(h => {
        if (h.key && h.value) headers[h.key] = h.value;
      });

      // Add auth header
      if (data.auth.type === 'bearer' && data.auth.bearer) {
        headers.Authorization = `Bearer ${data.auth.bearer}`;
      }

      const startTime = Date.now();
      const fetchResponse = await fetch(data.url, {
        method: data.method,
        headers,
        body: data.method !== 'GET' && data.body.type !== 'none' ? data.body.raw : undefined
      });

      const responseData = await fetchResponse.json();
      const endTime = Date.now();

      const result = {
        status: fetchResponse.status,
        statusText: fetchResponse.statusText,
        data: responseData,
        headers: Object.fromEntries(fetchResponse.headers.entries()),
        time: endTime - startTime,
        size: JSON.stringify(responseData).length
      };

      setResponse(result);

      // Update activity
      setActivities(prev => 
        prev.map(a => 
          a.id === newActivity.id 
            ? { ...a, message: `${data.method} ${data.url} - ${result.status} ${result.statusText}`, status: 'success' }
            : a
        )
      );

    } catch (error) {
      setResponse({ error: error.message, status: 0 });
      setActivities(prev => 
        prev.map(a => 
          a.id === newActivity.id 
            ? { ...a, message: `${data.method} ${data.url} - Error: ${error.message}`, status: 'error' }
            : a
        )
      );
    } finally {
      setIsLoading(false);
    }
  };

  const handleFileSelect = (file) => {
    // Simulate loading a saved request
    console.log('Selected file:', file);
    // You would load the request data from the file here
  };

  const handleTestSuiteSelect = (suite) => {
    console.log('Selected test suite:', suite);
    // You would run the test suite here
  };

  const handleClearActivities = () => {
    setActivities([]);
  };

  return (
    <div className="h-screen bg-background flex flex-col">
      {/* Header */}
      <div className="bg-card border-b border-border px-4 py-3">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-semibold text-foreground">HydraLabs</h1>
          <div className="flex items-center gap-2">
            <button className="px-3 py-1 text-sm bg-muted text-muted-foreground rounded hover:bg-muted/80">
              <Save className="w-4 h-4 inline mr-1" />
              Save
            </button>
            <button className="px-3 py-1 text-sm bg-primary text-primary-foreground rounded hover:bg-primary/90">
              <Settings className="w-4 h-4 inline mr-1" />
              Settings
            </button>
          </div>
        </div>
      </div>

      {/* Main Layout */}
      <div className="flex-1 flex overflow-hidden">
        {/* Sidebar */}
        <SideExplorer 
          onFileSelect={handleFileSelect}
          onTestSuiteSelect={handleTestSuiteSelect}
        />

        {/* Main Content */}
        <div className="flex-1 flex flex-col">
          {/* Request Builder */}
          <div className="flex-1 min-h-0">
            <RequestBuilder 
              requestData={requestData}
              setRequestData={setRequestData}
              onSendRequest={handleSendRequest}
            />
          </div>

          {/* Response Viewer */}
          <div className="flex-1 min-h-0">
            <ResponseViewer 
              response={response}
              isLoading={isLoading}
            />
          </div>
        </div>
      </div>

      {/* Activity Panel */}
      <ActivityPanel 
        activities={activities}
        onClear={handleClearActivities}
      />
    </div>
  );
}
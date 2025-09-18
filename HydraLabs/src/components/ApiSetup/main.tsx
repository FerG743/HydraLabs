import React, { useState } from 'react';
import { 
  Play, 
  Save, 
  Copy, 
  Eye, 
  EyeOff, 
  Plus, 
  X, 
  Code, 
  FileText, 
  Settings,
  ChevronDown,
  ChevronRight,
  AlertCircle,
  CheckCircle,
  Clock,
  Zap,
  Globe,
  Lock,
  Database
} from 'lucide-react';

const ApiSetup = ({ 
  onSendRequest,
  onSaveRequest,
  className = '',
  initialData = {}
}) => {
  const [requestData, setRequestData] = useState({
    method: 'GET',
    url: '',
    headers: [{ key: '', value: '', enabled: true }],
    params: [{ key: '', value: '', enabled: true }],
    body: {
      type: 'none',
      raw: '',
      formData: [{ key: '', value: '', type: 'text', enabled: true }]
    },
    auth: {
      type: 'none',
      bearer: '',
      basic: { username: '', password: '' },
      apiKey: { key: '', value: '', addTo: 'header' }
    },
    tests: '',
    preRequest: '',
    ...initialData
  });

  const [isLoading, setIsLoading] = useState(false);
  const [expandedSections, setExpandedSections] = useState(['auth', 'body']);
  const [showPasswords, setShowPasswords] = useState({});

  const httpMethods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'];
  const methodColors = {
    GET: 'bg-green-100 text-green-700 border-green-200',
    POST: 'bg-blue-100 text-blue-700 border-blue-200',
    PUT: 'bg-orange-100 text-orange-700 border-orange-200',
    PATCH: 'bg-yellow-100 text-yellow-700 border-yellow-200',
    DELETE: 'bg-red-100 text-red-700 border-red-200',
    HEAD: 'bg-gray-100 text-gray-700 border-gray-200',
    OPTIONS: 'bg-purple-100 text-purple-700 border-purple-200'
  };

  const commonHeaders = [
    'Content-Type',
    'Authorization',
    'Accept',
    'User-Agent',
    'X-API-Key',
    'Cache-Control'
  ];

  const contentTypes = [
    'application/json',
    'application/xml',
    'application/x-www-form-urlencoded',
    'multipart/form-data',
    'text/plain'
  ];

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

  const addKeyValue = (type) => {
    const newItem = type === 'formData' 
      ? { key: '', value: '', type: 'text', enabled: true }
      : { key: '', value: '', enabled: true };
    
    setRequestData(prev => ({
      ...prev,
      [type]: [...prev[type], newItem]
    }));
  };

  const removeKeyValue = (type, index) => {
    setRequestData(prev => ({
      ...prev,
      [type]: prev[type].filter((_, i) => i !== index)
    }));
  };

  const updateKeyValue = (type, index, field, value) => {
    setRequestData(prev => ({
      ...prev,
      [type]: prev[type].map((item, i) => 
        i === index ? { ...item, [field]: value } : item
      )
    }));
  };

  const toggleSection = (section) => {
    setExpandedSections(prev =>
      prev.includes(section)
        ? prev.filter(s => s !== section)
        : [...prev, section]
    );
  };

  const handleSendRequest = async () => {
    setIsLoading(true);
    try {
      if (onSendRequest) {
        await onSendRequest(requestData);
      }
      await new Promise(resolve => setTimeout(resolve, 1000));
    } finally {
      setIsLoading(false);
    }
  };

  const QuickAuthSelector = () => (
    <div className="flex items-center gap-2">
      <Lock className="w-4 h-4 text-gray-500" />
      <select
        value={requestData.auth.type}
        onChange={(e) => updateRequestData('auth.type', e.target.value)}
        className="text-sm border-0 bg-transparent focus:ring-1 focus:ring-blue-500 rounded transition-all duration-200"
      >
        <option value="none">No Auth</option>
        <option value="bearer">Bearer</option>
        <option value="basic">Basic</option>
        <option value="api-key">API Key</option>
      </select>
      {requestData.auth.type !== 'none' && (
        <input
          type="password"
          placeholder={
            requestData.auth.type === 'bearer' ? 'Token' :
            requestData.auth.type === 'basic' ? 'user:pass' : 'API Key'
          }
          className="text-sm px-2 py-1 border border-gray-200 rounded focus:ring-1 focus:ring-blue-500 transition-all duration-200 min-w-[120px]"
          value={
            requestData.auth.type === 'bearer' ? requestData.auth.bearer :
            requestData.auth.type === 'basic' ? `${requestData.auth.basic.username}:${requestData.auth.basic.password}` :
            requestData.auth.apiKey.value
          }
          onChange={(e) => {
            if (requestData.auth.type === 'bearer') {
              updateRequestData('auth.bearer', e.target.value);
            } else if (requestData.auth.type === 'basic') {
              const [username, password] = e.target.value.split(':');
              updateRequestData('auth.basic.username', username || '');
              updateRequestData('auth.basic.password', password || '');
            } else {
              updateRequestData('auth.apiKey.value', e.target.value);
            }
          }}
        />
      )}
    </div>
  );

  const QuickHeaders = () => (
    <div className="flex flex-wrap items-center gap-2">
      <FileText className="w-4 h-4 text-gray-500" />
      <select
        onChange={(e) => {
          if (e.target.value) {
            const existingIndex = requestData.headers.findIndex(h => h.key === e.target.value);
            if (existingIndex === -1) {
              addKeyValue('headers');
              updateKeyValue('headers', requestData.headers.length, 'key', e.target.value);
              if (e.target.value === 'Content-Type') {
                updateKeyValue('headers', requestData.headers.length, 'value', 'application/json');
              }
            }
            e.target.value = '';
          }
        }}
        className="text-sm border-0 bg-transparent focus:ring-1 focus:ring-blue-500 rounded"
      >
        <option value="">Add Header</option>
        {commonHeaders.map(header => (
          <option key={header} value={header}>{header}</option>
        ))}
      </select>
      {requestData.headers.filter(h => h.key && h.enabled).map((header, i) => (
        <span key={i} className="inline-flex items-center gap-1 px-2 py-1 bg-blue-50 text-blue-700 text-sm rounded animate-scale-in">
          {header.key}
          <button
            onClick={() => removeKeyValue('headers', requestData.headers.findIndex(h => h === header))}
            className="hover:text-blue-900 transition-colors duration-200"
          >
            <X className="w-3 h-3" />
          </button>
        </span>
      ))}
    </div>
  );

  const QuickParams = () => (
    <div className="flex flex-wrap items-center gap-2">
      <Settings className="w-4 h-4 text-gray-500" />
      <button
        onClick={() => addKeyValue('params')}
        className="text-sm text-blue-600 hover:text-blue-700 transition-colors duration-200"
      >
        Add Param
      </button>
      {requestData.params.filter(p => p.key && p.enabled).map((param, i) => (
        <span key={i} className="inline-flex items-center gap-1 px-2 py-1 bg-green-50 text-green-700 text-sm rounded animate-scale-in">
          {param.key}={param.value}
          <button
            onClick={() => removeKeyValue('params', requestData.params.findIndex(p => p === param))}
            className="hover:text-green-900 transition-colors duration-200"
          >
            <X className="w-3 h-3" />
          </button>
        </span>
      ))}
    </div>
  );

  return (
    <div className={`bg-white rounded-lg border border-gray-200 ${className}`}>
      {/* Compact Header with URL */}
      <div className="p-4 space-y-3">
        {/* Main Request Line */}
        <div className="flex items-center gap-3 animate-fade-in">
          <select
            value={requestData.method}
            onChange={(e) => updateRequestData('method', e.target.value)}
            className={`px-3 py-2 text-sm font-medium rounded border transition-all duration-200 ${
              methodColors[requestData.method] || 'bg-gray-100 text-gray-700 border-gray-200'
            }`}
          >
            {httpMethods.map(method => (
              <option key={method} value={method}>{method}</option>
            ))}
          </select>
          
          <input
            type="text"
            placeholder="https://api.example.com/endpoint"
            value={requestData.url}
            onChange={(e) => updateRequestData('url', e.target.value)}
            className="flex-1 px-3 py-2 border border-gray-200 rounded focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all duration-300 focus:scale-[1.01]"
          />
          
          <button
            onClick={handleSendRequest}
            disabled={isLoading || !requestData.url}
            className={`flex items-center gap-2 px-6 py-2 text-white rounded transition-all duration-200 ${
              isLoading || !requestData.url
                ? 'bg-gray-400 cursor-not-allowed'
                : 'bg-blue-600 hover:bg-blue-700 hover:scale-105 active:scale-95'
            }`}
          >
            {isLoading ? (
              <>
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                Sending...
              </>
            ) : (
              <>
                <Play className="w-4 h-4" />
                Send
              </>
            )}
          </button>
        </div>

        {/* Quick Configuration Row */}
        <div className="flex items-center justify-between text-sm">
          <div className="flex items-center gap-6">
            <QuickAuthSelector />
            <QuickParams />
          </div>
          
          <div className="flex items-center gap-2">
            <button
              onClick={() => onSaveRequest && onSaveRequest(requestData)}
              className="flex items-center gap-1 px-3 py-1 text-gray-600 hover:text-gray-800 hover:bg-gray-100 rounded transition-all duration-200"
            >
              <Save className="w-4 h-4" />
              Save
            </button>
          </div>
        </div>

        {/* Quick Headers */}
        <QuickHeaders />
      </div>

      {/* Expandable Sections */}
      <div className="border-t border-gray-200">
        {/* Body Section - Most Common */}
        {(['POST', 'PUT', 'PATCH'].includes(requestData.method)) && (
          <div className="border-b border-gray-100">
            <button
              onClick={() => toggleSection('body')}
              className="w-full flex items-center gap-3 p-3 text-left hover:bg-gray-50 transition-all duration-200"
            >
              {expandedSections.includes('body') ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
              <Code className="w-4 h-4 text-gray-600" />
              <span className="font-medium">Request Body</span>
              {requestData.body.type !== 'none' && (
                <span className="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded">
                  {requestData.body.type}
                </span>
              )}
            </button>
            {expandedSections.includes('body') && (
              <div className="p-4 pt-0 animate-slide-down">
                <div className="flex gap-2 mb-3">
                  {['none', 'raw', 'form-data'].map(type => (
                    <button
                      key={type}
                      onClick={() => updateRequestData('body.type', type)}
                      className={`px-3 py-1 text-sm rounded transition-all duration-200 ${
                        requestData.body.type === type
                          ? 'bg-blue-100 text-blue-700'
                          : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                      }`}
                    >
                      {type === 'none' ? 'None' : type === 'raw' ? 'JSON/Raw' : 'Form Data'}
                    </button>
                  ))}
                </div>

                {requestData.body.type === 'raw' && (
                  <div className="animate-slide-down">
                    <div className="flex gap-2 mb-2">
                      <select
                        onChange={(e) => {
                          if (e.target.value) {
                            const ctHeader = requestData.headers.find(h => h.key === 'Content-Type');
                            if (ctHeader) {
                              ctHeader.value = e.target.value;
                            } else {
                              addKeyValue('headers');
                              updateKeyValue('headers', requestData.headers.length, 'key', 'Content-Type');
                              updateKeyValue('headers', requestData.headers.length, 'value', e.target.value);
                            }
                          }
                        }}
                        className="text-sm px-2 py-1 border border-gray-200 rounded focus:ring-1 focus:ring-blue-500"
                      >
                        <option value="">Content Type</option>
                        {contentTypes.map(type => (
                          <option key={type} value={type}>{type}</option>
                        ))}
                      </select>
                    </div>
                    <textarea
                      placeholder={`{\n  "name": "example",\n  "value": "data"\n}`}
                      value={requestData.body.raw}
                      onChange={(e) => updateRequestData('body.raw', e.target.value)}
                      rows={6}
                      className="w-full px-3 py-2 border border-gray-200 rounded focus:ring-2 focus:ring-blue-500 transition-all duration-200 font-mono text-sm"
                    />
                  </div>
                )}

                {requestData.body.type === 'form-data' && (
                  <div className="animate-slide-down space-y-2">
                    {requestData.body.formData.map((item, index) => (
                      <div key={index} className="flex items-center gap-2 animate-fade-in-up">
                        <input
                          type="text"
                          placeholder="Key"
                          value={item.key}
                          onChange={(e) => updateKeyValue('formData', index, 'key', e.target.value)}
                          className="flex-1 px-3 py-2 border border-gray-200 rounded focus:ring-1 focus:ring-blue-500 transition-all duration-200"
                        />
                        <input
                          type="text"
                          placeholder="Value"
                          value={item.value}
                          onChange={(e) => updateKeyValue('formData', index, 'value', e.target.value)}
                          className="flex-1 px-3 py-2 border border-gray-200 rounded focus:ring-1 focus:ring-blue-500 transition-all duration-200"
                        />
                        <button
                          onClick={() => removeKeyValue('formData', index)}
                          className="p-2 text-red-500 hover:text-red-700 transition-all duration-200 hover:scale-110"
                        >
                          <X className="w-4 h-4" />
                        </button>
                      </div>
                    ))}
                    <button
                      onClick={() => addKeyValue('formData')}
                      className="flex items-center gap-2 px-3 py-2 text-blue-600 hover:text-blue-700 hover:bg-blue-50 rounded transition-all duration-200"
                    >
                      <Plus className="w-4 h-4" />
                      Add Field
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* Advanced Section */}
        <div className="border-b border-gray-100">
          <button
            onClick={() => toggleSection('advanced')}
            className="w-full flex items-center gap-3 p-3 text-left hover:bg-gray-50 transition-all duration-200"
          >
            {expandedSections.includes('advanced') ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
            <Settings className="w-4 h-4 text-gray-600" />
            <span className="font-medium">Advanced</span>
            <span className="text-xs text-gray-500">Headers, Params, Scripts</span>
          </button>
          {expandedSections.includes('advanced') && (
            <div className="p-4 pt-0 animate-slide-down space-y-4">
              {/* Detailed Headers */}
              <div>
                <h4 className="font-medium text-sm mb-2">Headers</h4>
                <div className="space-y-2">
                  {requestData.headers.map((header, index) => (
                    <div key={index} className="flex items-center gap-2 animate-fade-in-up">
                      <input
                        type="checkbox"
                        checked={header.enabled}
                        onChange={(e) => updateKeyValue('headers', index, 'enabled', e.target.checked)}
                        className="w-4 h-4 text-blue-600 rounded"
                      />
                      <input
                        type="text"
                        placeholder="Header name"
                        value={header.key}
                        onChange={(e) => updateKeyValue('headers', index, 'key', e.target.value)}
                        className="flex-1 px-3 py-2 border border-gray-200 rounded focus:ring-1 focus:ring-blue-500 transition-all duration-200 text-sm"
                      />
                      <input
                        type="text"
                        placeholder="Value"
                        value={header.value}
                        onChange={(e) => updateKeyValue('headers', index, 'value', e.target.value)}
                        className="flex-1 px-3 py-2 border border-gray-200 rounded focus:ring-1 focus:ring-blue-500 transition-all duration-200 text-sm"
                      />
                      <button
                        onClick={() => removeKeyValue('headers', index)}
                        className="p-2 text-red-500 hover:text-red-700 transition-all duration-200"
                      >
                        <X className="w-4 h-4" />
                      </button>
                    </div>
                  ))}
                  <button
                    onClick={() => addKeyValue('headers')}
                    className="text-sm text-blue-600 hover:text-blue-700"
                  >
                    + Add Header
                  </button>
                </div>
              </div>

              {/* Tests */}
              <div>
                <h4 className="font-medium text-sm mb-2">Tests (JavaScript)</h4>
                <textarea
                  placeholder="pm.test('Status is 200', () => &#123; pm.response.to.have.status(200); &#125;);"
                  value={requestData.tests}
                  onChange={(e) => updateRequestData('tests', e.target.value)}
                  rows={4}
                  className="w-full px-3 py-2 border border-gray-200 rounded focus:ring-2 focus:ring-blue-500 transition-all duration-200 font-mono text-sm"
                />
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default ApiSetup;
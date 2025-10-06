// utils/apiConstants.js
export const HTTP_METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'];

export const METHOD_COLORS = {
  GET: 'bg-green-100 text-green-700 border-green-200',
  POST: 'bg-blue-100 text-blue-700 border-blue-200',
  PUT: 'bg-orange-100 text-orange-700 border-orange-200',
  PATCH: 'bg-yellow-100 text-yellow-700 border-yellow-200',
  DELETE: 'bg-red-100 text-red-700 border-red-200',
  HEAD: 'bg-muted text-foreground border-border',
  OPTIONS: 'bg-purple-100 text-purple-700 border-purple-200'
};

export const COMMON_HEADERS = [
  'Content-Type',
  'Authorization',
  'Accept',
  'User-Agent',
  'X-API-Key',
  'Cache-Control',
  'X-Requested-With',
  'Accept-Language',
  'Accept-Encoding'
];

export const CONTENT_TYPES = [
  'application/json',
  'application/xml',
  'application/x-www-form-urlencoded',
  'multipart/form-data',
  'text/plain',
  'text/html',
  'text/csv'
];

export const AUTH_TYPES = [
  { value: 'none', label: 'No Auth' },
  { value: 'bearer', label: 'Bearer Token' },
  { value: 'basic', label: 'Basic Auth' },
  { value: 'api-key', label: 'API Key' }
];

export const BODY_TYPES = [
  { value: 'none', label: 'None' },
  { value: 'raw', label: 'JSON/Raw' },
  { value: 'form-data', label: 'Form Data' },
  { value: 'x-www-form-urlencoded', label: 'URL Encoded' }
];

// utils/apiHelpers.js
export const getMethodColor = (method) => {
  return METHOD_COLORS[method] || 'bg-muted text-foreground border-border';
};

export const hasRequestBody = (method) => {
  return ['POST', 'PUT', 'PATCH'].includes(method);
};

export const validateUrl = (url) => {
  try {
    new URL(url);
    return { isValid: true, error: null };
  } catch (error) {
    return { isValid: false, error: 'Please enter a valid URL' };
  }
};

export const formatAuthForDisplay = (auth) => {
  switch (auth.type) {
    case 'bearer':
      return auth.bearer ? `Bearer ${auth.bearer.substring(0, 10)}...` : '';
    case 'basic':
      return auth.basic.username ? `${auth.basic.username}:***` : '';
    case 'api-key':
      return auth.apiKey.key ? `${auth.apiKey.key}: ***` : '';
    default:
      return '';
  }
};

export const buildRequestUrl = (baseUrl, params) => {
  if (!baseUrl || !params?.length) return baseUrl;
  
  const enabledParams = params.filter(p => p.key && p.enabled);
  if (!enabledParams.length) return baseUrl;
  
  const url = new URL(baseUrl);
  enabledParams.forEach(param => {
    url.searchParams.set(param.key, param.value);
  });
  
  return url.toString();
};

export const getDefaultRequestTemplate = (type = 'json') => {
  const templates = {
    json: '{\n  "name": "example",\n  "value": "data"\n}',
    xml: '<?xml version="1.0"?>\n<root>\n  <item>value</item>\n</root>',
    text: 'Plain text content'
  };
  
  return templates[type] || templates.json;
};

import { CheckCircle, XCircle, AlertCircle } from 'lucide-react';

export const getStatusIcon = (status) => {
  switch(status) {
    case 'passed': return <CheckCircle className="w-4 h-4 text-green-500" />;
    case 'failed': return <XCircle className="w-4 h-4 text-red-500" />;
    case 'running': return <AlertCircle className="w-4 h-4 text-yellow-500 animate-pulse" />;
    default: return <AlertCircle className="w-4 h-4 text-muted-foreground" />;
  }
};

export const getTypeColor = (type) => {
  switch(type) {
    case 'unit': return 'bg-blue-100 text-blue-800';
    case 'integration': return 'bg-purple-100 text-purple-800';
    case 'functional': return 'bg-green-100 text-green-800';
    case 'performance': return 'bg-orange-100 text-orange-800';
    default: return 'bg-muted text-gray-800';
  }
};

export const defaultTestSuites = [
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
  },
  {
    id: 'test-suite-4',
    name: 'UI Component Tests',
    type: 'unit',
    status: 'passed',
    tests: 24,
    passed: 24,
    failed: 0,
    lastRun: '5 min ago'
  }
];

export const defaultFileTree = [
  {
    name: 'test-suites',
    type: 'folder',
    children: [
      { name: 'login.spec.js', type: 'file', size: '2.4 KB' },
      { name: 'registration.spec.js', type: 'file', size: '1.8 KB' },
      { name: 'auth-flow.spec.js', type: 'file', size: '3.2 KB' }
    ]
  },
  {
    name: 'api-tests',
    type: 'folder', 
    children: [
      { name: 'user-api.test.js', type: 'file', size: '4.1 KB' },
      { name: 'auth-api.test.js', type: 'file', size: '2.9 KB' },
      { name: 'product-api.test.js', type: 'file', size: '5.3 KB' }
    ]
  },
  {
    name: 'performance',
    type: 'folder',
    children: [
      { name: 'load-test.js', type: 'file', size: '1.5 KB' },
      { name: 'stress-test.js', type: 'file', size: '2.1 KB' }
    ]
  },
  {
    name: 'config',
    type: 'folder',
    children: [
      { name: 'test.config.js', type: 'file', size: '890 B' },
      { name: 'jest.config.js', type: 'file', size: '1.2 KB' }
    ]
  }
];
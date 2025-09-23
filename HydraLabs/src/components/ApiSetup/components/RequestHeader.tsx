import React from 'react';
import { Save, Copy } from 'lucide-react';
import { Button } from '@/components/ui/button';
import MethodSelector from './MethodSelector';
import UrlInput from './UrlInput';
import SendButton from './SendButton';
import QuickAuthSelector from './QuickAuthSelector';
import QuickParams from './QuickParams';
import QuickHeaders from './QuickHeaders';

const RequestHeader = ({
  requestData,
  isLoading,
  onUpdate,
  onSendRequest,
  onSaveRequest,
  headerActions,
  paramActions
}) => {
  const { method, url, auth, headers, params } = requestData;
  
  const isValidUrl = url && url.trim().length > 0;

  const handleCopyUrl = () => {
    navigator.clipboard.writeText(url);
    // Could add a toast notification here
  };

  return (
    <div className="p-4 space-y-3">
      {/* Main Request Line */}
      <div className="flex items-center gap-3 animate-fade-in">
        <MethodSelector
          method={method}
          onChange={(newMethod) => onUpdate('method', newMethod)}
        />
        
        <UrlInput
          value={url}
          onChange={(newUrl) => onUpdate('url', newUrl)}
          placeholder="https://api.example.com/endpoint"
        />
        
        <SendButton
          onSend={onSendRequest}
          isLoading={isLoading}
          disabled={!isValidUrl}
        />
      </div>

      {/* Quick Configuration Row */}
      <div className="flex items-center justify-between text-sm">
        <div className="flex items-center gap-6">
          <QuickAuthSelector
            auth={auth}
            onUpdate={onUpdate}
          />
          
          <QuickParams
            params={params}
            onAdd={paramActions.onAdd}
            onRemove={paramActions.onRemove}
          />
        </div>
        
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleCopyUrl}
          >
            <Copy className="w-4 h-4 mr-2" />
            Copy
          </Button>
          
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onSaveRequest && onSaveRequest(requestData)}
          >
            <Save className="w-4 h-4 mr-2" />
            Save
          </Button>
        </div>
      </div>

      {/* Quick Headers */}
      <QuickHeaders
        headers={headers}
        onAdd={headerActions.onAdd}
        onRemove={headerActions.onRemove}
        onUpdate={headerActions.onUpdate}
      />
    </div>
  );
}import React from 'react';
import { Save, Copy } from 'lucide-react';
import { Button } from '@/components/ui/button';
import MethodSelector from './MethodSelector';
import UrlInput from './UrlInput';
import SendButton from './SendButton';
import QuickAuthSelector from './QuickAuthSelector';
import QuickParams from './QuickParams';
import QuickHeaders from './QuickHeaders';

const RequestHeader = ({
  requestData,
  isLoading,
  onUpdate,
  onSendRequest,
  onSaveRequest,
  headerActions,
  paramActions
}) => {
  const { method, url, auth, headers, params } = requestData;
  
  const isValidUrl = url && url.trim().length > 0;

  const handleCopyUrl = () => {
    navigator.clipboard.writeText(url);
    // Could add a toast notification here
  };

  return (
    <div className="p-4 space-y-3">
      {/* Main Request Line */}
      <div className="flex items-center gap-3 animate-fade-in">
        <MethodSelector
          method={method}
          onChange={(newMethod) => onUpdate('method', newMethod)}
        />
        
        <UrlInput
          value={url}
          onChange={(newUrl) => onUpdate('url', newUrl)}
          placeholder="https://api.example.com/endpoint"
        />
        
        <SendButton
          onSend={onSendRequest}
          isLoading={isLoading}
          disabled={!isValidUrl}
        />
      </div>

      {/* Quick Configuration Row */}
      <div className="flex items-center justify-between text-sm">
        <div className="flex items-center gap-6">
          <QuickAuthSelector
            auth={auth}
            onUpdate={onUpdate}
          />
          
          <QuickParams
            params={params}
            onAdd={paramActions.onAdd}
            onRemove={paramActions.onRemove}
          />
        </div>
        
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleCopyUrl}
          >
            <Copy className="w-4 h-4 mr-2" />
            Copy
          </Button>
          
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onSaveRequest && onSaveRequest(requestData)}
          >
            <Save className="w-4 h-4 mr-2" />
            Save
          </Button>
        </div>
      </div>

      {/* Quick Headers */}
      <QuickHeaders
        headers={headers}
        onAdd={headerActions.onAdd}
        onRemove={headerActions.onRemove}
        onUpdate={headerActions.onUpdate}
      />
    </div>
  );
};

export default RequestHeader;
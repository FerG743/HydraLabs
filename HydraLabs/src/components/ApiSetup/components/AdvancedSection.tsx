import React from 'react';
import { Settings } from 'lucide-react';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import KeyValueEditor from './KeyValueEditor';
import ExpandableSection from './ExpandibleSection';

const AdvancedSection = ({
  isExpanded,
  onToggle,
  headers,
  params,
  tests,
  preRequest,
  onUpdate,
  headerActions,
  paramActions
}) => {
  return (
    <ExpandableSection
      title="Advanced"
      icon={Settings}
      isExpanded={isExpanded}
      onToggle={onToggle}
      subtitle="Headers, Parameters, Scripts"
    >
      <div className="space-y-6">
        {/* Headers Section */}
        <div>
          <div className="flex items-center justify-between mb-3">
            <h4 className="font-medium text-sm">Headers</h4>
            <span className="text-xs text-muted-foreground">
              {headers.filter(h => h.enabled && h.key).length} enabled
            </span>
          </div>
          
          <KeyValueEditor
            items={headers}
            onAdd={headerActions.onAdd}
            onRemove={headerActions.onRemove}
            onUpdate={headerActions.onUpdate}
            onClear={headerActions.onClear}
            placeholder={{ key: 'Header name', value: 'Header value' }}
            showEnabled={true}
            showType={false}
            addButtonText="Add Header"
            emptyMessage="No headers configured"
          />
        </div>

        {/* Parameters Section */}
        <div>
          <div className="flex items-center justify-between mb-3">
            <h4 className="font-medium text-sm">URL Parameters</h4>
            <span className="text-xs text-muted-foreground">
              {params.filter(p => p.enabled && p.key).length} enabled
            </span>
          </div>
          
          <KeyValueEditor
            items={params}
            onAdd={paramActions.onAdd}
            onRemove={paramActions.onRemove}
            onUpdate={paramActions.onUpdate}
            onClear={paramActions.onClear}
            placeholder={{ key: 'Parameter name', value: 'Parameter value' }}
            showEnabled={true}
            showType={false}
            addButtonText="Add Parameter"
            emptyMessage="No parameters configured"
          />
        </div>

        {/* Pre-request Script */}
        <div>
          <h4 className="font-medium text-sm mb-2">Pre-request Script</h4>
          <Textarea
            placeholder="// JavaScript code to run before the request&#10;// Example:&#10;// pm.environment.set('timestamp', Date.now());"
            value={preRequest}
            onChange={(e) => onUpdate('preRequest', e.target.value)}
            rows={4}
            className="font-mono text-sm resize-y"
          />
          <div className="text-xs text-muted-foreground mt-1">
            JavaScript code executed before sending the request
          </div>
        </div>

        {/* Tests Script */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <h4 className="font-medium text-sm">Tests</h4>
            <Button variant="ghost" size="sm" className="text-xs h-6">
              Templates
            </Button>
          </div>
          
          <Textarea
            placeholder="// Write tests for the response&#10;// Example:&#10;// pm.test('Status is 200', () => {&#10;//   pm.response.to.have.status(200);&#10;// });"
            value={tests}
            onChange={(e) => onUpdate('tests', e.target.value)}
            rows={6}
            className="font-mono text-sm resize-y"
          />
          <div className="text-xs text-muted-foreground mt-1">
            JavaScript test assertions to validate the response
          </div>
        </div>
      </div>
    </ExpandableSection>
  );
};

export default AdvancedSection;
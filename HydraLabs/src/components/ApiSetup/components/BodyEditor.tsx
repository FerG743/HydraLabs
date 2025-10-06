import React from 'react';
import { Code } from 'lucide-react';
import { BODY_TYPES, CONTENT_TYPES } from '@/utils/apiConstants';
import { getDefaultRequestTemplate } from '@/utils/apiConstants';
import KeyValueEditor from './KeyValueEditor';
import ExpandableSection from './ExpandibleSection';
const BodyEditor = ({
  body,
  method,
  isExpanded,
  onToggle,
  onUpdate,
  onAddFormData,
  onRemoveFormData,
  onUpdateFormData,
  onClearFormData,
  hasRequestBody
}) => {
  const handleContentTypeChange = (contentType) => {
    // This would need to be handled by the parent component
    // since it involves updating headers
    console.log('Content type changed to:', contentType);
  };

  const handleBodyTypeChange = (newType) => {
    onUpdate('body.type', newType);
    
    // Set default content based on type
    if (newType === 'raw' && !body.raw) {
      onUpdate('body.raw', getDefaultRequestTemplate('json'));
    }
  };

  if (!hasRequestBody) {
    return null;
  }

  return (
    <ExpandableSection
      title="Request Body"
      icon={Code}
      isExpanded={isExpanded}
      onToggle={onToggle}
      badge={body.type !== 'none' ? body.type : undefined}
    >
      <div className="space-y-4">
        {/* Body Type Selector */}
        <div className="flex gap-2">
          {BODY_TYPES.map(type => (
            <button
              key={type.value}
              onClick={() => handleBodyTypeChange(type.value)}
              className={`px-3 py-1 text-sm rounded transition-all duration-200 ${
                body.type === type.value
                  ? 'bg-blue-100 text-blue-700'
                  : 'bg-muted text-muted-foreground hover:bg-border'
              }`}
            >
              {type.label}
            </button>
          ))}
        </div>

        {/* Raw Body Editor */}
        {body.type === 'raw' && (
          <div className="animate-slide-down space-y-3">
            <div className="flex gap-2">
              <select
                onChange={(e) => handleContentTypeChange(e.target.value)}
                className="text-sm px-3 py-1 border border-border rounded focus:ring-1 focus:ring-blue-500"
                defaultValue=""
              >
                <option value="">Select Content Type</option>
                {CONTENT_TYPES.map(type => (
                  <option key={type} value={type}>{type}</option>
                ))}
              </select>
            </div>
            
            <div className="relative">
              <textarea
                placeholder={getDefaultRequestTemplate('json')}
                value={body.raw}
                onChange={(e) => onUpdate('body.raw', e.target.value)}
                rows={8}
                className="w-full px-3 py-3 border border-border rounded focus:ring-2 focus:ring-blue-500 transition-all duration-200 font-mono text-sm resize-y"
              />
              <div className="absolute bottom-2 right-2 text-xs text-muted-foreground">
                {body.raw.length} characters
              </div>
            </div>
          </div>
        )}

        {/* Form Data Editor */}
        {body.type === 'form-data' && (
          <div className="animate-slide-down">
            <KeyValueEditor
              items={body.formData}
              onAdd={onAddFormData}
              onRemove={onRemoveFormData}
              onUpdate={onUpdateFormData}
              onClear={onClearFormData}
              placeholder={{ key: 'Field name', value: 'Field value' }}
              showEnabled={true}
              showType={true}
              addButtonText="Add Field"
              emptyMessage="No form fields added yet"
            />
          </div>
        )}

        {/* URL Encoded Editor */}
        {body.type === 'x-www-form-urlencoded' && (
          <div className="animate-slide-down">
            <KeyValueEditor
              items={body.formData}
              onAdd={onAddFormData}
              onRemove={onRemoveFormData}
              onUpdate={onUpdateFormData}
              onClear={onClearFormData}
              placeholder={{ key: 'Parameter', value: 'Value' }}
              showEnabled={true}
              showType={false}
              addButtonText="Add Parameter"
              emptyMessage="No parameters added yet"
            />
          </div>
        )}

        {/* None State */}
        {body.type === 'none' && (
          <div className="text-center py-6 text-muted-foreground">
            <Code className="w-8 h-8 mx-auto mb-2 opacity-50" />
            <div className="text-sm">No request body</div>
            <div className="text-xs mt-1">Select a body type above to add content</div>
          </div>
        )}
      </div>
    </ExpandableSection>
  );
};

export default BodyEditor;
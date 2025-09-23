import React from 'react';
import { Card } from '@/components/ui/card';
import { useApiRequest } from './hooks/useApiRequest';
import { useKeyValuePairs } from './hooks/useKeyValuePairs';
import { useExpandableItems } from './hooks/useExpandableItems';
import { hasRequestBody } from './utils/apiHelpers';
import RequestHeader from './RequestHeader';
import BodyEditor from './components/BodyEditor';
import AdvancedSection from './components/AdvancedSection';

const ApiSetup = ({ 
  onSendRequest,
  onSaveRequest,
  className = '',
  initialData = {}
}) => {
  const {
    requestData,
    updateRequestData,
    isLoading,
    setIsLoading
  } = useApiRequest(initialData);

  const { isExpanded: isExpandedSection, toggle: toggleSection } = useExpandableItems(['body']);

  // Headers management
  const headerActions = useKeyValuePairs(
    requestData,
    updateRequestData,
    'headers'
  );

  // Parameters management
  const paramActions = useKeyValuePairs(
    requestData,
    updateRequestData,
    'params'
  );

  // Form data management
  const formDataActions = useKeyValuePairs(
    requestData,
    (path, value) => updateRequestData(`body.${path}`, value),
    'formData'
  );

  const handleSendRequest = async () => {
    setIsLoading(true);
    try {
      if (onSendRequest) {
        await onSendRequest(requestData);
      }
      // Simulate request delay
      await new Promise(resolve => setTimeout(resolve, 1000));
    } catch (error) {
      console.error('Request failed:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const showBodySection = hasRequestBody(requestData.method);

  return (
    <Card className={className}>
      {/* Request Header */}
      <RequestHeader
        requestData={requestData}
        isLoading={isLoading}
        onUpdate={updateRequestData}
        onSendRequest={handleSendRequest}
        onSaveRequest={onSaveRequest}
        headerActions={headerActions}
        paramActions={paramActions}
      />

      {/* Expandable Sections */}
      <div className="border-t">
        {/* Body Section */}
        {showBodySection && (
          <BodyEditor
            body={requestData.body}
            method={requestData.method}
            isExpanded={isExpandedSection('body')}
            onToggle={() => toggleSection('body')}
            onUpdate={updateRequestData}
            onAddFormData={() => formDataActions.addKeyValue('formData')}
            onRemoveFormData={formDataActions.removeKeyValue}
            onUpdateFormData={formDataActions.updateKeyValue}
            onClearFormData={formDataActions.clearAll}
            hasRequestBody={showBodySection}
          />
        )}

        {/* Advanced Section */}
        <AdvancedSection
          isExpanded={isExpandedSection('advanced')}
          onToggle={() => toggleSection('advanced')}
          headers={requestData.headers}
          params={requestData.params}
          tests={requestData.tests}
          preRequest={requestData.preRequest}
          onUpdate={updateRequestData}
          headerActions={headerActions}
          paramActions={paramActions}
        />
      </div>
    </Card>
  );
};

export default ApiSetup;
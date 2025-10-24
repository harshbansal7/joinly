/**
 * Citation component for displaying grounded content with hoverable citations
 */

'use client';

import React, { useState } from 'react';
import { ExternalLink, Search, CheckCircle } from 'lucide-react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import type { GroundingMetadata, GroundingChunk } from '@/lib/api';

interface CitationProps {
  text: string;
  groundingMetadata?: GroundingMetadata;
  className?: string;
}

interface CitationModalProps {
  chunk: GroundingChunk;
  citationNumber: number;
  supportedText?: string[];
  queries?: string[];
}

function CitationModal({ chunk, citationNumber, supportedText, queries }: CitationModalProps) {
  const domain = new URL(chunk.web.uri).hostname;

  return (
    <DialogContent className="sm:max-w-[600px]">
      <DialogHeader>
        <DialogTitle className="flex items-center gap-2">
          <div className="flex items-center justify-center w-6 h-6 bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300 rounded-full text-sm font-medium">
            {citationNumber}
          </div>
          Source Details
        </DialogTitle>
      </DialogHeader>
      
      <div className="space-y-4">
        {/* Source Information */}
        <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4">
          <div className="flex items-start justify-between">
            <div className="flex-1 min-w-0">
              <h3 className="font-semibold text-gray-900 dark:text-white truncate">
                {chunk.web.title || 'Untitled'}
              </h3>
              <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                {domain}
              </p>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => window.open(chunk.web.uri, '_blank')}
              className="ml-4 flex-shrink-0"
            >
              <ExternalLink className="h-4 w-4 mr-2" />
              Open
            </Button>
          </div>
        </div>

        {/* Search Queries */}
        {queries && queries.length > 0 && (
          <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4">
            <h4 className="font-medium text-gray-900 dark:text-white mb-3 flex items-center gap-2">
              <Search className="h-4 w-4" />
              Related Search Queries
            </h4>
            <div className="flex flex-wrap gap-2">
              {queries.map((query, index) => (
                <Badge key={index} variant="secondary" className="text-xs">
                  {query}
                </Badge>
              ))}
            </div>
          </div>
        )}

        {/* Supported Content */}
        {supportedText && supportedText.length > 0 && (
          <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4">
            <h4 className="font-medium text-gray-900 dark:text-white mb-3 flex items-center gap-2">
              <CheckCircle className="h-4 w-4" />
              Content Supported by This Source
            </h4>
            <ScrollArea className="max-h-32">
              <div className="space-y-2">
                {supportedText.map((text, index) => (
                  <div key={index} className="text-sm text-gray-700 dark:text-gray-300 p-2 bg-gray-50 dark:bg-gray-900/50 rounded">
                    &ldquo;{text}&rdquo;
                  </div>
                ))}
              </div>
            </ScrollArea>
          </div>
        )}

        {/* Source URL */}
        <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4">
          <h4 className="font-medium text-gray-900 dark:text-white mb-2">Full URL</h4>
          <div className="bg-gray-100 dark:bg-gray-800 rounded p-2">
            <code className="text-xs text-gray-700 dark:text-gray-300 break-all">
              {chunk.web.uri}
            </code>
          </div>
        </div>
      </div>
    </DialogContent>
  );
}

export function Citation({ text, groundingMetadata, className }: CitationProps) {
  const [hoveredCitation, setHoveredCitation] = useState<number | null>(null);

  if (!groundingMetadata || !groundingMetadata.grounding_chunks) {
    return <span className={className}>{text}</span>;
  }

  // Parse citation links from text
  const citationRegex = /\[(\d+)\]\((https?:\/\/[^\)]+)\)/g;
  const parts: (string | { type: 'citation', number: number, url: string })[] = [];
  let lastIndex = 0;
  let match;

  while ((match = citationRegex.exec(text)) !== null) {
    // Add text before citation
    if (match.index > lastIndex) {
      parts.push(text.slice(lastIndex, match.index));
    }
    
    // Add citation
    parts.push({
      type: 'citation',
      number: parseInt(match[1]),
      url: match[2]
    });
    
    lastIndex = match.index + match[0].length;
  }

  // Add remaining text
  if (lastIndex < text.length) {
    parts.push(text.slice(lastIndex));
  }

  return (
    <span className={className}>
      {parts.map((part, index) => {
        if (typeof part === 'string') {
          return <span key={index}>{part}</span>;
        }

        const chunk = groundingMetadata.grounding_chunks?.[part.number - 1];
        if (!chunk) {
          return <span key={index}>[{part.number}]</span>;
        }

        // Find supported text for this citation
        const supportedText = groundingMetadata.grounding_supports
          ?.filter(support => support.grounding_chunk_indices.includes(part.number - 1))
          .map(support => support.segment.text) || [];

        return (
          <Dialog key={index}>
            <DialogTrigger asChild>
              <button
                className={`inline-flex items-center justify-center w-5 h-5 text-xs font-medium rounded-full transition-colors
                  ${hoveredCitation === part.number 
                    ? 'bg-blue-600 text-white scale-110' 
                    : 'bg-blue-100 text-blue-800 hover:bg-blue-200 dark:bg-blue-900 dark:text-blue-300 dark:hover:bg-blue-800'
                  } mx-0.5 cursor-pointer transform`}
                onMouseEnter={() => setHoveredCitation(part.number)}
                onMouseLeave={() => setHoveredCitation(null)}
              >
                {part.number}
              </button>
            </DialogTrigger>
            <CitationModal
              chunk={chunk}
              citationNumber={part.number}
              supportedText={supportedText}
              queries={groundingMetadata.web_search_queries}
            />
          </Dialog>
        );
      })}
    </span>
  );
}
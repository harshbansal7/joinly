/**
 * Enhanced Analysis Section with grounding support and improved design
 */

'use client';

import React, { useState } from 'react';
import {
  CheckCircle,
  Target,
  Users,
  Clock,
  AlertTriangle,
  Loader2,
  Search,
  TrendingUp,
  Brain,
  Lightbulb,
  MessageSquare,
  Zap,
  FileText,
  Microscope,
  ArrowRight,
  UserCheck,
  Calendar
} from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Citation } from '@/components/ui/citation';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import type { AnalysisData, ActionItem } from '@/lib/api';

interface AnalysisSectionProps {
  analysis: AnalysisData;
  isLoading?: boolean;
  onRefresh?: () => void;
  onDownload?: () => void;
}

interface StatCardProps {
  icon: React.ReactNode;
  title: string;
  value: string | number;
  subtitle?: string;
  color: 'blue' | 'green' | 'purple' | 'orange' | 'pink';
  trend?: { value: string; isPositive: boolean };
}

function StatCard({ icon, title, value, subtitle, color, trend }: StatCardProps) {
  const colorClasses = {
    blue: 'bg-blue-50 dark:bg-blue-950/20 border-blue-200 dark:border-blue-800 text-blue-600 dark:text-blue-400',
    green: 'bg-green-50 dark:bg-green-950/20 border-green-200 dark:border-green-800 text-green-600 dark:text-green-400',
    purple: 'bg-purple-50 dark:bg-purple-950/20 border-purple-200 dark:border-purple-800 text-purple-600 dark:text-purple-400',
    orange: 'bg-orange-50 dark:bg-orange-950/20 border-orange-200 dark:border-orange-800 text-orange-600 dark:text-orange-400',
    pink: 'bg-pink-50 dark:bg-pink-950/20 border-pink-200 dark:border-pink-800 text-pink-600 dark:text-pink-400'
  };

  const textColorClasses = {
    blue: 'text-blue-900 dark:text-blue-100',
    green: 'text-green-900 dark:text-green-100',
    purple: 'text-purple-900 dark:text-purple-100',
    orange: 'text-orange-900 dark:text-orange-100',
    pink: 'text-pink-900 dark:text-pink-100'
  };

  return (
    <div className={`p-4 rounded-xl border ${colorClasses[color]}`}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-white/50 dark:bg-gray-900/50">
            {icon}
          </div>
          <div>
            <p className={`text-sm font-medium ${textColorClasses[color]}`}>
              {title}
            </p>
            <p className={`text-2xl font-bold ${textColorClasses[color]}`}>
              {value}
            </p>
            {subtitle && (
              <p className={`text-xs opacity-75 ${textColorClasses[color]}`}>
                {subtitle}
              </p>
            )}
          </div>
        </div>
        {trend && (
          <div className={`flex items-center gap-1 text-xs font-medium ${
            trend.isPositive ? 'text-green-600' : 'text-red-600'
          }`}>
            <TrendingUp className={`h-3 w-3 ${trend.isPositive ? '' : 'rotate-180'}`} />
            {trend.value}
          </div>
        )}
      </div>
    </div>
  );
}

function GroundingIndicator({ hasGrounding }: { hasGrounding: boolean }) {
  return (
    <div className="flex items-center gap-2">
      <div className={`w-2 h-2 rounded-full ${hasGrounding ? 'bg-green-500' : 'bg-gray-400'}`} />
      <span className="text-xs text-gray-500 dark:text-gray-400">
        {hasGrounding ? 'AI-Grounded' : 'Standard Analysis'}
      </span>
      {hasGrounding && (
        <Search className="h-3 w-3 text-green-500" />
      )}
    </div>
  );
}

function EnhancedSummaryCard({ analysis }: { analysis: AnalysisData }) {
  const hasGrounding = !!analysis.grounded_summary?.grounding_metadata;
  const displayText = hasGrounding 
    ? analysis.grounded_summary?.text_with_citations || analysis.summary
    : analysis.summary;

  return (
    <Card className="relative overflow-hidden">
      <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-blue-100/20 to-purple-100/20 dark:from-blue-900/10 dark:to-purple-900/10 rounded-bl-full" />
      <CardHeader className="pb-4">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-3 text-lg">
            <div className="p-2 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg">
              <Brain className="h-5 w-5 text-white" />
            </div>
            Meeting Summary
          </CardTitle>
          <GroundingIndicator hasGrounding={hasGrounding} />
        </div>
      </CardHeader>
      <CardContent>
        <div className="prose prose-sm max-w-none dark:prose-invert">
          {hasGrounding ? (
            <Citation 
              text={displayText}
              groundingMetadata={analysis.grounded_summary?.grounding_metadata}
              className="text-gray-700 dark:text-gray-300 leading-relaxed"
            />
          ) : (
            <p className="text-gray-700 dark:text-gray-300 leading-relaxed">
              {analysis.summary}
            </p>
          )}
        </div>
        
        {hasGrounding && analysis.grounded_summary?.grounding_metadata?.grounding_chunks && (
          <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-800">
            <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
              <Search className="h-4 w-4" />
              <span>
                Based on {analysis.grounded_summary.grounding_metadata.grounding_chunks?.length} verified sources
              </span>
              <span className="text-xs bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300 px-2 py-1 rounded-full">
                AI-Verified
              </span>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function EnhancedKeyPointsCard({ analysis }: { analysis: AnalysisData }) {
  const hasGrounding = !!analysis.grounded_key_points?.grounding_metadata;
  const keyPoints = analysis.key_points || [];

  return (
    <Card className="relative overflow-hidden">
      <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-green-100/20 to-emerald-100/20 dark:from-green-900/10 dark:to-emerald-900/10 rounded-bl-full" />
      <CardHeader className="pb-4">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-3 text-lg">
            <div className="p-2 bg-gradient-to-br from-green-500 to-emerald-600 rounded-lg">
              <Lightbulb className="h-5 w-5 text-white" />
            </div>
            Key Insights
          </CardTitle>
          <GroundingIndicator hasGrounding={hasGrounding} />
        </div>
      </CardHeader>
      <CardContent>
        {keyPoints.length > 0 ? (
          <div className="space-y-3">
            {keyPoints.map((point, index) => (
              <div key={index} className="flex items-start gap-3 p-3 rounded-lg bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-800">
                <div className="flex items-center justify-center w-6 h-6 bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300 rounded-full text-sm font-medium flex-shrink-0 mt-0.5">
                  {index + 1}
                </div>
                <div className="flex-1">
                  {hasGrounding && analysis.grounded_key_points?.grounding_metadata ? (
                    <Citation 
                      text={point}
                      groundingMetadata={analysis.grounded_key_points.grounding_metadata}
                      className="text-gray-700 dark:text-gray-300"
                    />
                  ) : (
                    <span className="text-gray-700 dark:text-gray-300">
                      {point}
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-6 text-gray-500 dark:text-gray-400">
            No key points identified yet
          </div>
        )}
        
        {hasGrounding && analysis.grounded_key_points?.grounding_metadata?.grounding_chunks && (
          <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-800">
            <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
              <Search className="h-4 w-4" />
              <span>
                {analysis.grounded_key_points.grounding_metadata.grounding_chunks?.length} sources verified these insights
              </span>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ActionItemsCard({ actionItems }: { actionItems: ActionItem[] }) {
  const [expandedItems] = useState<Set<string>>(new Set());

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'high': return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300';
      case 'medium': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300';
      case 'low': return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300';
      default: return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300';
    }
  };

  const formatPriority = (priority: string) => {
    switch (priority) {
      case 'high': return 'High';
      case 'medium': return 'Medium';
      case 'low': return 'Low';
      default: return priority.charAt(0).toUpperCase() + priority.slice(1);
    }
  };

  const getPriorityIcon = (priority: string) => {
    switch (priority) {
      case 'high': return <AlertTriangle className="h-3 w-3" />;
      case 'medium': return <Clock className="h-3 w-3" />;
      case 'low': return <CheckCircle className="h-3 w-3" />;
      default: return <Target className="h-3 w-3" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'bg-green-100 text-green-800 border-green-300 dark:bg-green-900 dark:text-green-300';
      case 'in_progress': return 'bg-blue-100 text-blue-800 border-blue-300 dark:bg-blue-900 dark:text-blue-300';
      default: return 'bg-gray-100 text-gray-800 border-gray-300 dark:bg-gray-900 dark:text-gray-300';
    }
  };

  const formatStatus = (status: string) => {
    switch (status) {
      case 'completed': return 'Completed';
      case 'in_progress': return 'In Progress';
      case 'pending': return 'Pending';
      default: return status ? status.charAt(0).toUpperCase() + status.slice(1) : 'Pending';
    }
  };

  const getTypeColor = (type?: string) => {
    switch (type) {
      case 'task': return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300';
      case 'research': return 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300';
      case 'investigation': return 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-300';
      case 'follow-up': return 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300';
      case 'decision': return 'bg-pink-100 text-pink-800 dark:bg-pink-900 dark:text-pink-300';
      default: return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300';
    }
  };

  const formatType = (type?: string) => {
    switch (type) {
      case 'task': return 'Task';
      case 'research': return 'Research';
      case 'investigation': return 'Investigation';
      case 'follow-up': return 'Follow-up';
      case 'decision': return 'Decision';
      default: return type ? type.charAt(0).toUpperCase() + type.slice(1) : 'Task';
    }
  };

  const getTypeIcon = (type?: string) => {
    switch (type) {
      case 'task': return <CheckCircle className="h-3 w-3" />;
      case 'research': return <Search className="h-3 w-3" />;
      case 'investigation': return <Microscope className="h-3 w-3" />;
      case 'follow-up': return <ArrowRight className="h-3 w-3" />;
      case 'decision': return <UserCheck className="h-3 w-3" />;
      default: return <Target className="h-3 w-3" />;
    }
  };

  return (
    <Card className="relative overflow-hidden">
      <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-orange-100/20 to-red-100/20 dark:from-orange-900/10 dark:to-red-900/10 rounded-bl-full" />
      <CardHeader>
        <CardTitle className="flex items-center gap-3 text-lg">
          <div className="p-2 bg-gradient-to-br from-orange-500 to-red-600 rounded-lg shadow-sm">
            <Target className="h-5 w-5 text-white" />
          </div>
          Action Items
          <Badge variant="secondary" className="ml-auto bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300">
            {actionItems.length} {actionItems.length === 1 ? 'item' : 'items'}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent>
        {actionItems.length > 0 ? (
          <div className="space-y-3">
            {actionItems.map((item) => (
              <Collapsible key={item.id}>
                <div className="border border-gray-200 dark:border-gray-700 rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-shadow duration-200 bg-white dark:bg-gray-900">
                  <CollapsibleTrigger asChild>
                    <div className="p-5 hover:bg-gray-50/50 dark:hover:bg-gray-800/50 cursor-pointer transition-colors duration-200">
                      <div className="flex items-start justify-between">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-2">
                            <span className="font-medium text-gray-900 dark:text-white">
                              {item.description}
                            </span>
                          </div>
                          <div className="flex items-center gap-2 flex-wrap">
                            <Badge className={`${getPriorityColor(item.priority)} px-2 py-1 text-xs font-semibold rounded-full flex items-center gap-1`}>
                              {getPriorityIcon(item.priority)}
                              <span>{formatPriority(item.priority)}</span>
                            </Badge>
                            {item.status && item.status.trim() !== '' && (
                              <Badge variant="outline" className={`${getStatusColor(item.status)} px-2 py-1 text-xs border-2 rounded-full`}>
                                {formatStatus(item.status)}
                              </Badge>
                            )}
                            {item.type && (
                              <Badge className={`${getTypeColor(item.type)} px-2 py-1 text-xs rounded-full flex items-center gap-1`}>
                                {getTypeIcon(item.type)}
                                <span className="opacity-80">Type:</span>
                                <span className="font-medium">{formatType(item.type)}</span>
                              </Badge>
                            )}
                            {item.assignee && (
                              <Badge variant="outline" className="px-2 py-1 text-xs rounded-full border-2 bg-white/50 dark:bg-gray-800/50 max-w-xs truncate">
                                <Users className="h-3 w-3 mr-1 opacity-70" />
                                <span className="truncate">{item.assignee}</span>
                              </Badge>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                  </CollapsibleTrigger>
                  <CollapsibleContent>
                    <div className="px-5 pb-5 bg-gradient-to-r from-gray-50/50 to-white/50 dark:from-gray-800/30 dark:to-gray-900/30 border-t border-gray-100 dark:border-gray-700">
                      <div className="grid grid-cols-2 gap-4 text-sm">
                        {/* {item.due_date && (
                          <div>
                            <span className="font-medium text-gray-600 dark:text-gray-400">Due Date:</span>
                            <p className="text-gray-900 dark:text-white">
                              {new Date(item.due_date).toLocaleDateString()}
                            </p>
                          </div>
                        )} */}
                        <div>
                          <span className="font-medium text-gray-600 dark:text-gray-400">Created:</span>
                          <p className="text-gray-900 dark:text-white">
                            {new Date(item.created_at).toLocaleDateString()}
                          </p>
                        </div>
                      </div>
                    </div>
                  </CollapsibleContent>
                </div>
              </Collapsible>
            ))}
          </div>
        ) : (
          <div className="text-center py-6 text-gray-500 dark:text-gray-400">
            No action items identified yet
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export function AnalysisSection({ analysis, isLoading }: AnalysisSectionProps) {
  // Note: isLoading is now handled at the button level, not here
  // This component always shows the analysis content when available

  const duration = analysis.duration_minutes || 0;
  const hours = Math.floor(duration / 60);
  const minutes = Math.floor(duration % 60);

  return (
    <div className="space-y-6">
      {/* Stats Overview */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard
          icon={<Clock className="h-5 w-5" />}
          title="Duration"
          value={`${hours}h ${minutes}m`}
          color="blue"
        />
        <StatCard
          icon={<Users className="h-5 w-5" />}
          title="Participants"
          value={analysis.participants?.length || 0}
          color="green"
        />
        <StatCard
          icon={<MessageSquare className="h-5 w-5" />}
          title="Words Spoken"
          value={analysis.word_count?.toLocaleString() || 0}
          color="purple"
        />
        <StatCard
          icon={<Target className="h-5 w-5" />}
          title="Action Items"
          value={analysis.action_items?.length || 0}
          color="orange"
        />
      </div>

      {/* Main Analysis Content */}
      <Tabs defaultValue="overview" className="w-full">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="overview" className="flex items-center gap-2">
            <Brain className="h-4 w-4" />
            Overview
          </TabsTrigger>
          <TabsTrigger value="actions" className="flex items-center gap-2">
            <Target className="h-4 w-4" />
            Actions
          </TabsTrigger>
          <TabsTrigger value="topics" className="flex items-center gap-2">
            <MessageSquare className="h-4 w-4" />
            Topics
          </TabsTrigger>
          <TabsTrigger value="insights" className="flex items-center gap-2">
            <Zap className="h-4 w-4" />
            Insights
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6 mt-6">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <EnhancedSummaryCard analysis={analysis} />
            <EnhancedKeyPointsCard analysis={analysis} />
          </div>
        </TabsContent>

        <TabsContent value="actions" className="mt-6">
          <ActionItemsCard actionItems={analysis.action_items || []} />
        </TabsContent>

        <TabsContent value="topics" className="mt-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-3 text-lg">
                <div className="p-2 bg-gradient-to-br from-purple-500 to-indigo-600 rounded-lg">
                  <Users className="h-5 w-5 text-white" />
                </div>
                Discussion Topics
              </CardTitle>
            </CardHeader>
            <CardContent>
              {analysis.topics && analysis.topics.length > 0 ? (
                <div className="space-y-4">
                  {analysis.topics.map((topic, index) => (
                    <div key={index} className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                      <div className="flex items-start justify-between mb-3">
                        <h4 className="font-semibold text-gray-900 dark:text-white">
                          {topic.topic}
                        </h4>
                        <Badge variant="outline" className="text-xs">
                          {topic.duration}m
                        </Badge>
                      </div>
                      <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
                        {topic.summary}
                      </p>
                      {topic.participants && topic.participants.length > 0 && (
                        <div className="flex items-center gap-2">
                          <span className="text-xs font-medium text-gray-500">Participants:</span>
                          <div className="flex gap-1">
                            {topic.participants.map((participant, pIndex) => (
                              <Badge key={pIndex} variant="secondary" className="text-xs">
                                {participant}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-6 text-gray-500 dark:text-gray-400">
                  No discussion topics identified yet
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="insights" className="mt-6">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Sentiment Analysis */}
            {analysis.sentiment && (
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-3 text-lg">
                    <div className="p-2 bg-gradient-to-br from-pink-500 to-rose-600 rounded-lg">
                      <AlertTriangle className="h-5 w-5 text-white" />
                    </div>
                    Meeting Sentiment
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center gap-3 mb-4">
                    <Badge
                      className={`text-sm px-3 py-1 ${
                        analysis.sentiment === 'positive' ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300' :
                        analysis.sentiment === 'negative' ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300' :
                        analysis.sentiment === 'mixed' ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300' :
                        'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300'
                      }`}
                    >
                      {analysis.sentiment.charAt(0).toUpperCase() + analysis.sentiment.slice(1)}
                    </Badge>
                  </div>
                  {analysis.keywords && analysis.keywords.length > 0 && (
                    <div>
                      <h4 className="font-medium text-gray-900 dark:text-white mb-2">Key Themes</h4>
                      <div className="flex flex-wrap gap-2">
                        {analysis.keywords.slice(0, 8).map((keyword, index) => (
                          <Badge key={index} variant="outline" className="text-xs">
                            {keyword}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>
            )}

            {/* Participant Overview */}
            {analysis.participants && analysis.participants.length > 0 && (
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-3 text-lg">
                    <div className="p-2 bg-gradient-to-br from-indigo-500 to-blue-600 rounded-lg">
                      <Users className="h-5 w-5 text-white" />
                    </div>
                    Participants
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    {analysis.participants.map((participant, index) => (
                      <div key={index} className="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-900/50 rounded-lg">
                        <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center text-white text-sm font-medium">
                          {participant.charAt(0).toUpperCase()}
                        </div>
                        <span className="font-medium text-gray-900 dark:text-white">
                          {participant}
                        </span>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
/**
 * Individual agent details page with comprehensive information and controls.
 */

'use client';

import { useEffect, useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useAgentStore, useUIStore } from '@/lib/store';
import { agentsApi } from '@/lib/api';
import {
  Play,
  Square,
  Trash2,
  RefreshCw,
  Download,
  Bot,
  Globe,
  Zap,
  Mic,
  MessageSquare,
  Activity,
  Clock,
  Settings,
  CheckCircle,
  AlertCircle,
  Loader2,
  Copy,
  ExternalLink,
  Eye,
  FileText,
  Users,
  Target,
  AlertTriangle,
} from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Layout } from '@/components/layout/layout';
import type { AnalysisData } from '@/lib/api';

export default function AgentDetailsPage() {
  const params = useParams();
  const router = useRouter();
  const agentId = params.id as string;

  const { agents, removeAgent, updateAgent, connectSessionWebSocket, disconnectSessionWebSocket } = useAgentStore();
  const { addNotification } = useUIStore();

  const [logs, setLogs] = useState<{ timestamp: string; level: string; message: string }[]>([]);
  const [isLoadingLogs, setIsLoadingLogs] = useState(false);
  const [logFilter, setLogFilter] = useState<string>('all');

  // Analysis state for analyst mode agents
  const [analysis, setAnalysis] = useState<AnalysisData | null>(null);
  const [isLoadingAnalysis, setIsLoadingAnalysis] = useState(false);
  const [formattedAnalysis, setFormattedAnalysis] = useState<string>('');

  const agent = agents.find(a => a.id === agentId);

  // Helper function to safely parse URLs
  const getUrlInfo = (url: string) => {
    try {
      const urlObj = new URL(url);
      return {
        hostname: urlObj.hostname,
        pathname: urlObj.pathname.slice(1)
      };
    } catch {
      return {
        hostname: 'Invalid URL',
        pathname: url || 'Invalid URL'
      };
    }
  };

  // Filter logs based on selected level
  const filteredLogs = logs.filter(log => {
    if (logFilter === 'all') return true;
    return log.level === logFilter;
  });

  const loadLogs = useCallback(async () => {
    setIsLoadingLogs(true);
    try {
      const response = await agentsApi.getLogs(agentId, 100);
      setLogs(response.data.logs || []);
    } catch (error) {
      console.error('Failed to load logs:', error);
    } finally {
      setIsLoadingLogs(false);
    }
  }, [agentId]);

  const loadAnalysis = useCallback(async () => {
    if (agent?.config.conversation_mode !== 'analyst') return;

    setIsLoadingAnalysis(true);
    try {
      const response = await agentsApi.getAnalysis(agentId);
      setAnalysis(response.data);

      // Also load formatted analysis
      const formattedResponse = await agentsApi.getFormattedAnalysis(agentId);
      setFormattedAnalysis(formattedResponse.data);
    } catch (error) {
      console.error('Failed to load analysis:', error);
      setAnalysis(null);
      setFormattedAnalysis('');
    } finally {
      setIsLoadingAnalysis(false);
    }
  }, [agentId, agent]);

  const downloadAnalysis = () => {
    if (!formattedAnalysis) return;

    const blob = new Blob([formattedAnalysis], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `meeting-analysis-${agentId}.md`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const formatLogEntry = (log: { timestamp: string; level: string; message: string }): string => {
    const timestamp = new Date(log.timestamp).toLocaleString();
    return `[${timestamp}] ${log.level.toUpperCase()}: ${log.message}`;
  };

  useEffect(() => {
    loadLogs();
    if (agent?.config.conversation_mode === 'analyst') {
      loadAnalysis();
    }
  }, [loadLogs, loadAnalysis, agent]);

  // Sync logs from agent state (updated via session WebSocket)
  useEffect(() => {
    if (agent && agent.logs) {
      setLogs(agent.logs);
    }
  }, [agent]);

  // Connect to session WebSocket for real-time updates
  useEffect(() => {
    connectSessionWebSocket((message) => {
      console.log('Agent details WebSocket message:', message);

      // Only process messages for this specific agent
      if (message.agent_id === agentId) {
        if (message.type === 'log') {
          const newLog = {
            timestamp: (message.data.timestamp as string) || new Date().toISOString(),
            level: (message.data.level as string) || 'info',
            message: (message.data.message as string) || 'Unknown log message'
          };
          setLogs(prevLogs => [...prevLogs, newLog]);
        }
      }
    });

    return () => {
      disconnectSessionWebSocket();
    };
  }, [connectSessionWebSocket, disconnectSessionWebSocket, agentId]);

  const downloadLogs = () => {
    const logText = filteredLogs.map(formatLogEntry).join('\n');
    const blob = new Blob([logText], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `agent-${agentId}-logs.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300';
      case 'error':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300';
      case 'starting':
      case 'stopping':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running':
        return <CheckCircle className="h-4 w-4 text-green-600" />;
      case 'error':
        return <AlertCircle className="h-4 w-4 text-red-600" />;
      case 'starting':
      case 'stopping':
        return <Loader2 className="h-4 w-4 text-blue-600 animate-spin" />;
      default:
        return <Square className="h-4 w-4 text-gray-400" />;
    }
  };

  const formatUptime = (startedAt?: string) => {
    if (!startedAt) return 'Never started';

    const start = new Date(startedAt);
    const now = new Date();
    const diff = now.getTime() - start.getTime();

    const hours = Math.floor(diff / (1000 * 60 * 60));
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));

    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const handleStart = async () => {
    if (!agent) return;

    try {
      // Update local state optimistically
      const optimisticAgent = {
        id: agent.id,
        config: agent.config,
        status: 'starting' as const,
        created_at: agent.created_at,
        started_at: agent.started_at,
        stopped_at: agent.stopped_at,
        error_message: agent.error_message,
        process_id: agent.process_id,
        logs: agent.logs
      };
      updateAgent(optimisticAgent);

      await agentsApi.start(agentId);
      addNotification({
        type: 'success',
        title: 'Agent Started',
        message: `${agent.config.name || 'Agent'} has been started successfully`
      });
    } catch {
      // Revert optimistic update on error
      const revertedAgent = {
        id: agent.id,
        config: agent.config,
        status: 'stopped' as const,
        created_at: agent.created_at,
        started_at: agent.started_at,
        stopped_at: agent.stopped_at,
        error_message: agent.error_message,
        process_id: agent.process_id,
        logs: agent.logs
      };
      updateAgent(revertedAgent);
      addNotification({
        type: 'error',
        title: 'Failed to Start Agent',
        message: 'An error occurred while starting the agent'
      });
    }
  };

  const handleStop = async () => {
    if (!agent) return;

    try {
      // Update local state optimistically
      const optimisticAgent = {
        id: agent.id,
        config: agent.config,
        status: 'stopping' as const,
        created_at: agent.created_at,
        started_at: agent.started_at,
        stopped_at: agent.stopped_at,
        error_message: agent.error_message,
        process_id: agent.process_id,
        logs: agent.logs
      };
      updateAgent(optimisticAgent);

      await agentsApi.stop(agentId);
      addNotification({
        type: 'success',
        title: 'Agent Stopped',
        message: `${agent.config.name || 'Agent'} has been stopped successfully`
      });
    } catch {
      // Revert optimistic update on error
      const revertedAgent = {
        id: agent.id,
        config: agent.config,
        status: agent.status,
        created_at: agent.created_at,
        started_at: agent.started_at,
        stopped_at: agent.stopped_at,
        error_message: agent.error_message,
        process_id: agent.process_id,
        logs: agent.logs
      };
      updateAgent(revertedAgent);
      addNotification({
        type: 'error',
        title: 'Failed to Stop Agent',
        message: 'An error occurred while stopping the agent'
      });
    }
  };

  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete this agent? This action cannot be undone.')) {
      return;
    }

    try {
      await removeAgent(agentId);
      addNotification({
        type: 'success',
        title: 'Agent Deleted',
        message: 'Agent has been deleted successfully'
      });
      router.push('/agents');
    } catch {
      addNotification({
        type: 'error',
        title: 'Failed to Delete Agent',
        message: 'An error occurred while deleting the agent'
      });
    }
  };

  if (!agent) {
    return (
      <Layout title="Agent Not Found">
        <div className="flex items-center justify-center min-h-96">
          <div className="text-center">
            <Bot className="h-12 w-12 text-gray-400 mx-auto mb-4" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">Agent Not Found</h1>
            <p className="text-gray-600 dark:text-gray-400 mb-4">The agent you&apos;re looking for doesn&apos;t exist or has been deleted.</p>
            <Button onClick={() => router.push('/agents')}>
              Back to Agents
            </Button>
          </div>
        </div>
      </Layout>
    );
  }

  return (
    <Layout title={`${agent.config.name} - Agent Details`}>
      <div className="space-y-6 max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center gap-4">
          <Button
            variant="outline"
            onClick={() => router.back()}
            className="flex items-center gap-2"
          >
            ← Back to Agents
          </Button>
        </div>

        {/* Hero Section */}
        <Card className="border-l-4 border-l-blue-500">
          <CardContent className="p-6">
            <div className="flex items-start justify-between">
              {/* Agent Info */}
              <div className="flex items-center gap-4 flex-1 min-w-0">
                <div className="relative">
                  <div className="p-3 bg-blue-50 dark:bg-blue-950/50 rounded-lg border border-blue-200 dark:border-blue-800">
                    <Bot className="h-8 w-8 text-blue-600 dark:text-blue-400" />
                  </div>
                  <div className={`absolute -bottom-1 -right-1 w-3 h-3 rounded-full border-2 border-white dark:border-gray-900 ${
                    agent.status === 'running' ? 'bg-green-500' :
                    agent.status === 'error' ? 'bg-red-500' :
                    agent.status === 'starting' || agent.status === 'stopping' ? 'bg-yellow-500' :
                    'bg-gray-400'
                  }`} />
                </div>
                <div className="flex-1 min-w-0">
                  <h1 className="text-2xl font-bold text-gray-900 dark:text-white truncate">
                    {agent.config.name}
                  </h1>
                  <div className="flex items-center gap-3 mt-2">
                    <Badge className={`${getStatusColor(agent.status)} gap-1.5 px-3 py-1 text-sm font-medium`}>
                      {getStatusIcon(agent.status)}
                      {agent.status.charAt(0).toUpperCase() + agent.status.slice(1)}
                    </Badge>
                    <div className="flex items-center gap-1 text-sm text-gray-600 dark:text-gray-400">
                      <Globe className="h-4 w-4" />
                      <span className="truncate">{getUrlInfo(agent.config.meeting_url).hostname}</span>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => copyToClipboard(agent.config.meeting_url)}
                        className="h-6 w-6 p-0 ml-1 hover:bg-gray-100 dark:hover:bg-gray-800"
                        title="Copy URL"
                      >
                        <Copy className="h-3 w-3" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => window.open(agent.config.meeting_url, '_blank')}
                        className="h-6 w-6 p-0 hover:bg-gray-100 dark:hover:bg-gray-800"
                        title="Open meeting"
                      >
                        <ExternalLink className="h-3 w-3" />
                      </Button>
                    </div>
                  </div>
                </div>
              </div>

              {/* Quick Stats */}
              <div className="hidden md:flex gap-6">
                <div className="text-center">
                  <div className="text-lg font-bold text-gray-900 dark:text-white">
                    {agent.status === 'running' ? formatUptime(agent.started_at) : '—'}
                  </div>
                  <div className="text-xs text-gray-600 dark:text-gray-400 mt-1">Uptime</div>
                </div>
                <div className="text-center">
                  <div className="text-lg font-bold text-gray-900 dark:text-white">
                    {new Date(agent.created_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}
                  </div>
                  <div className="text-xs text-gray-600 dark:text-gray-400 mt-1">Created</div>
                </div>
                <div className="text-center">
                  <div className="text-lg font-bold text-gray-900 dark:text-white">
                    {agent.config.llm_provider === 'google' ? 'Gemini' :
                     agent.config.llm_provider === 'openai' ? 'OpenAI' :
                     agent.config.llm_provider === 'anthropic' ? 'Claude' :
                     agent.config.llm_provider.charAt(0).toUpperCase() + agent.config.llm_provider.slice(1)}
                  </div>
                  <div className="text-xs text-gray-600 dark:text-gray-400 mt-1">AI Model</div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Action Buttons */}
        <div className="flex gap-3 flex-wrap">
          {agent.status === 'running' ? (
            <Button
              variant="outline"
              onClick={handleStop}
              className="flex-1 sm:flex-none"
            >
              <Square className="h-4 w-4 mr-2" />
              Stop Agent
            </Button>
          ) : agent.status === 'stopped' || agent.status === 'error' ? (
            <Button
              onClick={handleStart}
              className="flex-1 sm:flex-none"
            >
              <Play className="h-4 w-4 mr-2" />
              Start Agent
            </Button>
          ) : (
            <Button
              disabled
              className="flex-1 sm:flex-none"
            >
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              {agent.status === 'starting' ? 'Starting...' : 'Stopping...'}
            </Button>
          )}
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={agent.status === 'running'}
            className="flex-1 sm:flex-none"
          >
            <Trash2 className="h-4 w-4 mr-2" />
            Delete Agent
          </Button>
        </div>

        {/* Configuration */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings className="h-5 w-5" />
              Configuration
            </CardTitle>
            <CardDescription>
              Agent settings and capabilities
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* AI Intelligence */}
              <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4">
                <div className="flex items-center gap-3 mb-3">
                  <div className="p-2 bg-gray-100 dark:bg-gray-800 rounded-lg">
                    <Zap className="h-5 w-5 text-gray-600 dark:text-gray-400" />
                  </div>
                  <div>
                    <h3 className="text-base font-semibold text-gray-900 dark:text-white">AI Intelligence</h3>
                    <p className="text-sm text-gray-600 dark:text-gray-400">Language Model Configuration</p>
                  </div>
                </div>
                <div className="space-y-3">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Provider</p>
                      <p className="text-sm font-semibold text-gray-900 dark:text-white">
                        {agent.config.llm_provider === 'google' ? 'Google Gemini' :
                         agent.config.llm_provider === 'openai' ? 'OpenAI GPT' :
                         agent.config.llm_provider === 'anthropic' ? 'Anthropic Claude' :
                         agent.config.llm_provider.charAt(0).toUpperCase() + agent.config.llm_provider.slice(1)}
                      </p>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Model</p>
                      <p className="text-sm font-semibold text-gray-900 dark:text-white">{agent.config.llm_model}</p>
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Language</p>
                      <p className="text-sm font-semibold text-gray-900 dark:text-white uppercase">{agent.config.language}</p>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Style</p>
                      <p className="text-sm font-semibold text-gray-900 dark:text-white uppercase">{agent.config.prompt_style}</p>
                    </div>
                  </div>
                </div>
              </div>

              {/* Voice & Behavior */}
              <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4">
                <div className="flex items-center gap-3 mb-3">
                  <div className="p-2 bg-gray-100 dark:bg-gray-800 rounded-lg">
                    <Mic className="h-5 w-5 text-gray-600 dark:text-gray-400" />
                  </div>
                  <div>
                    <h3 className="text-base font-semibold text-gray-900 dark:text-white">Voice & Behavior</h3>
                    <p className="text-sm text-gray-600 dark:text-gray-400">Speech and Interaction Settings</p>
                  </div>
                </div>
                <div className="space-y-3">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Speech-to-Text</p>
                      <p className="text-sm font-semibold text-gray-900 dark:text-white capitalize">{agent.config.stt_provider}</p>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Text-to-Speech</p>
                      <p className="text-sm font-semibold text-gray-900 dark:text-white capitalize">{agent.config.tts_provider}</p>
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Auto-Join</p>
                      <div className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                        agent.config.auto_join
                          ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300'
                          : 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300'
                      }`}>
                        {agent.config.auto_join ? 'Enabled' : 'Disabled'}
                      </div>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Name Trigger</p>
                      <div className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                        agent.config.name_trigger
                          ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300'
                          : 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300'
                      }`}>
                        {agent.config.name_trigger ? 'Enabled' : 'Disabled'}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Logs, Status, and Analysis */}
        <Tabs defaultValue={agent.config.conversation_mode === 'analyst' ? 'analysis' : 'logs'} className="w-full">
          <TabsList className={`grid w-full ${agent.config.conversation_mode === 'analyst' ? 'grid-cols-3' : 'grid-cols-2'}`}>
            <TabsTrigger value="logs" className="flex items-center gap-2">
              <MessageSquare className="h-4 w-4" />
              Logs
            </TabsTrigger>
            <TabsTrigger value="status" className="flex items-center gap-2">
              <Activity className="h-4 w-4" />
              Status
            </TabsTrigger>
            {agent.config.conversation_mode === 'analyst' && (
              <TabsTrigger value="analysis" className="flex items-center gap-2">
                <Eye className="h-4 w-4" />
                Analysis
              </TabsTrigger>
            )}
          </TabsList>

          <TabsContent value="logs" className="space-y-4">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="flex items-center gap-2">
                      <MessageSquare className="h-5 w-5" />
                      Agent Logs
                    </CardTitle>
                    <CardDescription>
                      Real-time activity and debug information
                    </CardDescription>
                  </div>
                  <div className="flex gap-2 items-center">
                    {/* Log Filter */}
                    <Select value={logFilter} onValueChange={setLogFilter}>
                      <SelectTrigger className="w-32">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all">All Logs</SelectItem>
                        <SelectItem value="error">Errors</SelectItem>
                        <SelectItem value="warn">Warnings</SelectItem>
                        <SelectItem value="info">Info</SelectItem>
                        <SelectItem value="debug">Debug</SelectItem>
                      </SelectContent>
                    </Select>

                    <Button
                      variant="outline"
                      size="sm"
                      onClick={loadLogs}
                      disabled={isLoadingLogs}
                    >
                      <RefreshCw className={`h-4 w-4 mr-2 ${isLoadingLogs ? 'animate-spin' : ''}`} />
                      Refresh
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={downloadLogs}
                      disabled={filteredLogs.length === 0}
                    >
                      <Download className="h-4 w-4 mr-2" />
                      Download
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="h-96 w-full rounded-lg border bg-gray-50/50 dark:bg-gray-900/50">
                  {filteredLogs.length === 0 ? (
                    <div className="flex flex-col items-center justify-center h-full text-center">
                      <MessageSquare className="h-12 w-12 text-gray-400 mb-4" />
                      <p className="text-gray-600 dark:text-gray-400">
                        {isLoadingLogs ? 'Loading logs...' : 'No logs available yet'}
                      </p>
                    </div>
                  ) : (
                    <ScrollArea className="h-full w-full p-4">
                      <div className="space-y-2 font-mono text-sm">
                        {filteredLogs.map((log, index) => (
                          <div
                            key={index}
                            className={`p-3 rounded border-l-4 ${
                              log.level === 'error'
                                ? 'border-l-red-500 bg-red-50 dark:bg-red-950/20'
                                : log.level === 'warn'
                                ? 'border-l-yellow-500 bg-yellow-50 dark:bg-yellow-950/20'
                                : log.level === 'info'
                                ? 'border-l-blue-500 bg-blue-50 dark:bg-blue-950/20'
                                : 'border-l-gray-500 bg-gray-50 dark:bg-gray-950/20'
                            }`}
                          >
                            <div className="flex flex-col gap-2">
                              <div className="flex items-center gap-2 flex-shrink-0">
                                <span className="text-xs text-gray-600 dark:text-gray-400 font-sans">
                                  {new Date(log.timestamp).toLocaleTimeString()}
                                </span>
                                <Badge
                                  variant={
                                    log.level === 'error' ? 'destructive' :
                                    log.level === 'warn' ? 'secondary' :
                                    'outline'
                                  }
                                  className="text-xs px-1.5 py-0.5 h-5"
                                >
                                  {log.level.toUpperCase()}
                                </Badge>
                              </div>
                              <div className="text-sm font-mono leading-relaxed text-gray-900 dark:text-gray-100">
                                <pre className="whitespace-pre-wrap">
                                  {log.message}
                                </pre>
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                    </ScrollArea>
                  )}
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="status" className="space-y-6">
            {/* Status Overview */}
            <Card>
              <CardContent className="p-6">
                <div className="flex items-center gap-3 mb-6">
                  <div className="p-3 bg-gray-100 dark:bg-gray-800 rounded-lg">
                    <Activity className="h-6 w-6 text-gray-600 dark:text-gray-400" />
                  </div>
                  <div>
                    <h3 className="text-lg font-semibold text-gray-900 dark:text-white">System Status</h3>
                    <p className="text-sm text-gray-600 dark:text-gray-400">Technical information and runtime details</p>
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  {/* Agent Status */}
                  <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4">
                    <div className="flex items-center gap-3 mb-3">
                      <div className={`w-3 h-3 rounded-full ${
                        agent.status === 'running' ? 'bg-green-500' :
                        agent.status === 'error' ? 'bg-red-500' :
                        agent.status === 'starting' || agent.status === 'stopping' ? 'bg-yellow-500' :
                        'bg-gray-400'
                      }`} />
                      <h4 className="font-semibold text-gray-900 dark:text-white">Current Status</h4>
                    </div>
                    <p className="text-base font-bold capitalize text-gray-900 dark:text-white">
                      {agent.status === 'starting' && agent.config.auto_join ? 'Auto-starting...' : agent.status}
                    </p>
                    <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                      {agent.status === 'running' ? 'Agent is active in meeting' :
                       agent.status === 'error' ? 'Agent encountered an error' :
                       agent.status === 'starting' && agent.config.auto_join ? 'Agent is auto-starting due to auto-join setting' :
                       agent.status === 'starting' ? 'Agent is initializing' :
                       agent.status === 'stopping' ? 'Agent is shutting down' :
                       agent.status === 'created' && agent.config.auto_join ? 'Agent created, auto-starting...' :
                       'Agent is not active'}
                    </p>
                  </div>

                  {/* Agent ID */}
                  <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4">
                    <div className="flex items-center gap-3 mb-3">
                      <div className="w-3 h-3 bg-blue-500 rounded-full"></div>
                      <h4 className="font-semibold text-gray-900 dark:text-white">Agent ID</h4>
                    </div>
                    <div className="bg-gray-100 dark:bg-gray-800 p-2 rounded-lg">
                      <code className="text-sm font-mono text-gray-900 dark:text-white break-all">
                        {agent.id}
                      </code>
                    </div>
                  </div>

                  {/* Process ID */}
                  <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4">
                    <div className="flex items-center gap-3 mb-3">
                      <div className="w-3 h-3 bg-purple-500 rounded-full"></div>
                      <h4 className="font-semibold text-gray-900 dark:text-white">Process ID</h4>
                    </div>
                    <div className="bg-gray-100 dark:bg-gray-800 p-2 rounded-lg">
                      <code className="text-sm font-mono text-gray-900 dark:text-white">
                        {agent.process_id || 'N/A'}
                      </code>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Timeline */}
            <Card>
              <CardContent className="p-6">
                <div className="flex items-center gap-3 mb-6">
                  <div className="p-3 bg-gray-100 dark:bg-gray-800 rounded-lg">
                    <Clock className="h-6 w-6 text-gray-600 dark:text-gray-400" />
                  </div>
                  <div>
                    <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Timeline</h3>
                    <p className="text-sm text-gray-600 dark:text-gray-400">Important events and timestamps</p>
                  </div>
                </div>

                <div className="space-y-4">
                  {/* Creation */}
                  <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className="w-3 h-3 bg-green-500 rounded-full"></div>
                        <div>
                          <h4 className="font-semibold text-gray-900 dark:text-white">Created</h4>
                          <p className="text-sm text-gray-600 dark:text-gray-400">Agent was initialized</p>
                        </div>
                      </div>
                      <div className="text-right">
                        <p className="text-sm font-mono text-gray-900 dark:text-white">
                          {new Date(agent.created_at).toLocaleDateString()}
                        </p>
                        <p className="text-xs text-gray-600 dark:text-gray-400">
                          {new Date(agent.created_at).toLocaleTimeString()}
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Last Started */}
                  {agent.started_at && (
                    <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className="w-3 h-3 bg-blue-500 rounded-full"></div>
                          <div>
                            <h4 className="font-semibold text-gray-900 dark:text-white">Last Started</h4>
                            <p className="text-sm text-gray-600 dark:text-gray-400">Agent joined meeting</p>
                          </div>
                        </div>
                        <div className="text-right">
                          <p className="text-sm font-mono text-gray-900 dark:text-white">
                            {new Date(agent.started_at).toLocaleDateString()}
                          </p>
                          <p className="text-xs text-gray-600 dark:text-gray-400">
                            {new Date(agent.started_at).toLocaleTimeString()}
                          </p>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* Last Stopped */}
                  {agent.stopped_at && (
                    <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className="w-3 h-3 bg-red-500 rounded-full"></div>
                          <div>
                            <h4 className="font-semibold text-gray-900 dark:text-white">Last Stopped</h4>
                            <p className="text-sm text-gray-600 dark:text-gray-400">Agent left meeting</p>
                          </div>
                        </div>
                        <div className="text-right">
                          <p className="text-sm font-mono text-gray-900 dark:text-white">
                            {new Date(agent.stopped_at).toLocaleDateString()}
                          </p>
                          <p className="text-xs text-gray-600 dark:text-gray-400">
                            {new Date(agent.stopped_at).toLocaleTimeString()}
                          </p>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {agent.config.conversation_mode === 'analyst' && (
            <TabsContent value="analysis" className="space-y-6">
              {/* Analysis Overview */}
              <Card>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="flex items-center gap-2">
                        <Eye className="h-5 w-5" />
                        Meeting Analysis
                      </CardTitle>
                      <CardDescription>
                        Comprehensive meeting insights and structured notes
                      </CardDescription>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={loadAnalysis}
                        disabled={isLoadingAnalysis}
                      >
                        <RefreshCw className={`h-4 w-4 mr-2 ${isLoadingAnalysis ? 'animate-spin' : ''}`} />
                        Refresh
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={downloadAnalysis}
                        disabled={!formattedAnalysis || isLoadingAnalysis}
                      >
                        <Download className="h-4 w-4 mr-2" />
                        Download
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  {isLoadingAnalysis ? (
                    <div className="flex items-center justify-center py-8">
                      <Loader2 className="h-6 w-6 animate-spin mr-2" />
                      <span>Loading analysis...</span>
                    </div>
                  ) : analysis ? (
                    <div className="space-y-6">
                      {/* Analysis Summary Cards */}
                      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                        <div className="bg-blue-50 dark:bg-blue-950/20 p-4 rounded-lg border border-blue-200 dark:border-blue-800">
                          <div className="flex items-center gap-2">
                            <Clock className="h-5 w-5 text-blue-600" />
                            <div>
                              <p className="text-sm font-medium text-blue-900 dark:text-blue-100">Duration</p>
                              <p className="text-lg font-bold text-blue-900 dark:text-blue-100">
                                {Math.floor((analysis.duration_minutes || 0) / 60)}h {Math.floor((analysis.duration_minutes || 0) % 60)}m
                              </p>
                            </div>
                          </div>
                        </div>

                        <div className="bg-green-50 dark:bg-green-950/20 p-4 rounded-lg border border-green-200 dark:border-green-800">
                          <div className="flex items-center gap-2">
                            <Users className="h-5 w-5 text-green-600" />
                            <div>
                              <p className="text-sm font-medium text-green-900 dark:text-green-100">Participants</p>
                              <p className="text-lg font-bold text-green-900 dark:text-green-100">
                                {analysis.participants?.length || 0}
                              </p>
                            </div>
                          </div>
                        </div>

                        <div className="bg-purple-50 dark:bg-purple-950/20 p-4 rounded-lg border border-purple-200 dark:border-purple-800">
                          <div className="flex items-center gap-2">
                            <FileText className="h-5 w-5 text-purple-600" />
                            <div>
                              <p className="text-sm font-medium text-purple-900 dark:text-purple-100">Word Count</p>
                              <p className="text-lg font-bold text-purple-900 dark:text-purple-100">
                                {analysis.word_count || 0}
                              </p>
                            </div>
                          </div>
                        </div>

                        <div className="bg-orange-50 dark:bg-orange-950/20 p-4 rounded-lg border border-orange-200 dark:border-orange-800">
                          <div className="flex items-center gap-2">
                            <Target className="h-5 w-5 text-orange-600" />
                            <div>
                              <p className="text-sm font-medium text-orange-900 dark:text-orange-100">Action Items</p>
                              <p className="text-lg font-bold text-orange-900 dark:text-orange-100">
                                {analysis.action_items?.length || 0}
                              </p>
                            </div>
                          </div>
                        </div>
                      </div>

                      {/* Analysis Content */}
                      <div className="space-y-6">
                        {/* Summary */}
                        {analysis.summary && (
                          <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-6">
                            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                              <FileText className="h-5 w-5" />
                              Meeting Summary
                            </h3>
                            <div className="text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap">
                              {analysis.summary}
                            </div>
                          </div>
                        )}

                        {/* Key Points */}
                        {analysis.key_points && analysis.key_points.length > 0 && (
                          <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-6">
                            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                              <CheckCircle className="h-5 w-5" />
                              Key Points
                            </h3>
                            <ul className="space-y-2">
                              {analysis.key_points.map((point: string, index: number) => (
                                <li key={index} className="flex items-start gap-2 text-gray-700 dark:text-gray-300">
                                  <span className="text-blue-500 mt-1">•</span>
                                  <span>{point}</span>
                                </li>
                              ))}
                            </ul>
                          </div>
                        )}

                        {/* Action Items */}
                        {analysis.action_items && analysis.action_items.length > 0 && (
                          <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-6">
                            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                              <Target className="h-5 w-5" />
                              Action Items
                            </h3>
                            <div className="space-y-3">
                              {analysis.action_items.map((item, index: number) => (
                                <div key={index} className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 bg-gray-50 dark:bg-gray-900/50">
                                  <div className="flex items-start justify-between">
                                    <div className="flex-1">
                                      <div className="flex items-center gap-2 mb-2">
                                        <span className="font-medium text-gray-900 dark:text-white">{item.description}</span>
                                        <Badge
                                          className={`${
                                            item.priority === 'high' ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300' :
                                            item.priority === 'medium' ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300' :
                                            'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300'
                                          }`}
                                        >
                                          {item.priority || 'medium'}
                                        </Badge>
                                        <Badge
                                          variant="outline"
                                          className={`${
                                            item.status === 'completed' ? 'bg-green-100 text-green-800 border-green-300 dark:bg-green-900 dark:text-green-300' :
                                            item.status === 'in_progress' ? 'bg-blue-100 text-blue-800 border-blue-300 dark:bg-blue-900 dark:text-blue-300' :
                                            'bg-gray-100 text-gray-800 border-gray-300 dark:bg-gray-900 dark:text-gray-300'
                                          }`}
                                        >
                                          {item.status || 'pending'}
                                        </Badge>
                                      </div>
                                      {item.assignee && (
                                        <p className="text-sm text-gray-600 dark:text-gray-400 mb-1">
                                          <span className="font-medium">Assignee:</span> {item.assignee}
                                        </p>
                                      )}
                                      {item.due_date && (
                                        <p className="text-sm text-gray-600 dark:text-gray-400">
                                          <span className="font-medium">Due:</span> {new Date(item.due_date).toLocaleDateString()}
                                        </p>
                                      )}
                                    </div>
                                  </div>
                                </div>
                              ))}
                            </div>
                          </div>
                        )}

                        {/* Sentiment */}
                        {analysis.sentiment && (
                          <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-6">
                            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                              <AlertTriangle className="h-5 w-5" />
                              Meeting Sentiment
                            </h3>
                            <div className="flex items-center gap-3">
                              <Badge
                                className={`${
                                  analysis.sentiment === 'positive' ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300' :
                                  analysis.sentiment === 'negative' ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300' :
                                  analysis.sentiment === 'mixed' ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300' :
                                  'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300'
                                }`}
                              >
                                {analysis.sentiment}
                              </Badge>
                              {analysis.keywords && analysis.keywords.length > 0 && (
                                <div className="flex flex-wrap gap-1">
                                  {analysis.keywords.slice(0, 5).map((keyword: string, index: number) => (
                                    <Badge key={index} variant="outline" className="text-xs">
                                      {keyword}
                                    </Badge>
                                  ))}
                                </div>
                              )}
                            </div>
                          </div>
                        )}

                        {/* Topics */}
                        {analysis.topics && analysis.topics.length > 0 && (
                          <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-6">
                            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                              <Users className="h-5 w-5" />
                              Discussion Topics
                            </h3>
                            <div className="space-y-3">
                              {analysis.topics.map((topic, index: number) => (
                                <div key={index} className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                                  <div className="flex items-start justify-between">
                                    <div className="flex-1">
                                      <h4 className="font-medium text-gray-900 dark:text-white mb-2">{topic.topic}</h4>
                                      <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">{topic.summary}</p>
                                      <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-500">
                                        <span>{topic.duration} min</span>
                                        {topic.participants && (
                                          <span>{topic.participants.join(', ')}</span>
                                        )}
                                      </div>
                                    </div>
                                  </div>
                                </div>
                              ))}
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  ) : (
                    <div className="text-center py-8">
                      <Eye className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                      <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">No Analysis Available</h3>
                      <p className="text-gray-600 dark:text-gray-400 mb-4">
                        Analysis will be generated once the agent starts transcribing the meeting.
                      </p>
                      <Button onClick={loadAnalysis} disabled={isLoadingAnalysis}>
                        <RefreshCw className={`h-4 w-4 mr-2 ${isLoadingAnalysis ? 'animate-spin' : ''}`} />
                        Check for Analysis
                      </Button>
                    </div>
                  )}
                </CardContent>
              </Card>
            </TabsContent>
          )}
        </Tabs>
      </div>
    </Layout>
  );
}

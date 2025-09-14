/**
 * Modern agents management page with enhanced cards and features.
 */

'use client';

import { useEffect, useCallback } from 'react';
import Link from 'next/link';
import {
  Plus,
  Play,
  Square,
  Trash2,
  Eye,
  Clock,
  Zap,
  MessageSquare,
  Mic,
  Globe,
  Calendar,
  Activity,
  CheckCircle,
  AlertCircle,
  Loader2,
  Bot,
  MoreVertical
} from 'lucide-react';

import { Layout } from '@/components/layout/layout';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { useAgentStore, useUIStore } from '@/lib/store';
import { agentsApi, Agent, LogEntry } from '@/lib/api';

export default function AgentsPage() {
  const {
    agents,
    isLoading,
    setAgents,
    updateAgent,
    removeAgent,
    setLoading,
    connectSessionWebSocket,
    disconnectSessionWebSocket
  } = useAgentStore();

  const { addNotification } = useUIStore();

  const loadAgents = useCallback(async () => {
    setLoading(true);
    try {
      const response = await agentsApi.list();
      setAgents(response.data);
    } catch (error) {
      console.error('Failed to load agents:', error);
      addNotification({
        type: 'error',
        title: 'Failed to load agents',
        message: 'Could not load agents list. Please try again.',
      });
    } finally {
      setLoading(false);
    }
  }, [setAgents, addNotification, setLoading]);

  useEffect(() => {
    loadAgents();
  }, [loadAgents]);

  // Fallback polling for status updates (every 10 seconds) in case WebSocket fails
  useEffect(() => {
    const pollInterval = setInterval(async () => {
      try {
        const response = await agentsApi.list();
        const currentAgentIds = new Set(agents.map(a => a.id));
        const serverAgentIds = new Set(response.data.map(a => a.id));

        // Check for status changes or new/deleted agents
        const hasChanges = response.data.some((serverAgent) => {
          const localAgent = agents.find(a => a.id === serverAgent.id);
          return !localAgent ||
                 localAgent.status !== serverAgent.status ||
                 localAgent.logs.length !== serverAgent.logs.length;
        }) || currentAgentIds.size !== serverAgentIds.size;

        if (hasChanges) {
          console.log('🔄 Status changes detected, updating agents...');
          setAgents(response.data);
        }
      } catch (error) {
        console.error('Failed to poll agents:', error);
      }
    }, 10000); // Poll every 10 seconds

    return () => clearInterval(pollInterval);
  }, [agents, setAgents]);

  // Connect to session WebSocket for all agents
  useEffect(() => {
    connectSessionWebSocket((message) => {
      console.log('Session WebSocket message received:', message);

      if (message.type === 'status') {
        // Find and update the specific agent
        const agentToUpdate = agents.find(a => a.id === message.agent_id);
        if (agentToUpdate) {
          const updatedAgent = { 
            ...agentToUpdate, 
            status: message.data.status as Agent['status'],
            error_message: message.data.error ? message.data.error as string : agentToUpdate.error_message
          };
          updateAgent(updatedAgent);
        }
      } else if (message.type === 'log') {
        // Update agent logs - preserve existing logs and append new one
        const agentToUpdate = agents.find(a => a.id === message.agent_id);
        if (agentToUpdate) {
          const currentLogs = agentToUpdate.logs || [];
          const newLog: LogEntry = {
            timestamp: (message.data.timestamp as string) || new Date().toISOString(),
            level: (message.data.level as string) || 'info',
            message: (message.data.message as string) || 'Unknown log message'
          };
          const updatedAgent = {
            ...agentToUpdate,
            logs: [...currentLogs, newLog]
          };
          updateAgent(updatedAgent);
        }
      } else if (message.type === 'error') {
        // Update agent with error
        const agentToUpdate = agents.find(a => a.id === message.agent_id);
        if (agentToUpdate) {
          const updatedAgent = {
            ...agentToUpdate,
            status: 'error' as Agent['status'],
            error_message: message.data.message as string
          };
          updateAgent(updatedAgent);
        }
      }
    });

    return () => {
      // Cleanup session WebSocket
      disconnectSessionWebSocket();
    };
  }, [connectSessionWebSocket, disconnectSessionWebSocket, updateAgent, agents]);


  const handleStartAgent = async (agentId: string) => {
    try {
      await agentsApi.start(agentId);
      // The WebSocket will update the agent status
      addNotification({
        type: 'success',
        title: 'Agent started',
        message: 'Agent has been started successfully.',
      });
    } catch (error) {
      console.error('Failed to start agent:', error);
      addNotification({
        type: 'error',
        title: 'Failed to start agent',
        message: 'Could not start agent. Please try again.',
      });
    }
  };

  const handleStopAgent = async (agentId: string) => {
    try {
      await agentsApi.stop(agentId);
      // The WebSocket will update the agent status
      addNotification({
        type: 'success',
        title: 'Agent stopped',
        message: 'Agent has been stopped successfully.',
      });
    } catch (error) {
      console.error('Failed to stop agent:', error);
      addNotification({
        type: 'error',
        title: 'Failed to stop agent',
        message: 'Could not stop agent. Please try again.',
      });
    }
  };

  const handleDeleteAgent = async (agentId: string) => {
    try {
      await agentsApi.delete(agentId);
      removeAgent(agentId);
      addNotification({
        type: 'success',
        title: 'Agent deleted',
        message: 'Agent has been deleted successfully.',
      });
    } catch (error) {
      console.error('Failed to delete agent:', error);
      addNotification({
        type: 'error',
        title: 'Failed to delete agent',
        message: 'Could not delete agent. Please try again.',
      });
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

  return (
    <Layout title="Agents" subtitle="Create and manage your AI meeting assistants">
      <div className="space-y-8">
        {/* Header */}
        <div className="flex justify-between items-center">
          <div>
            <h2 className="text-3xl font-bold tracking-tight">Agents</h2>
            <p className="text-muted-foreground">
              Deploy AI agents that can participate in meetings autonomously
            </p>
          </div>
          <Link href="/agents/create">
            <Button size="lg" className="gap-2">
              <Plus className="h-5 w-5" />
              Create Agent
            </Button>
          </Link>
        </div>

        {/* Agents Grid */}
        {isLoading ? (
          <div className="flex justify-center items-center py-16">
            <div className="flex flex-col items-center gap-4">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              <p className="text-sm text-muted-foreground">Loading agents...</p>
            </div>
          </div>
        ) : agents.length === 0 ? (
          <Card className="border-dashed bg-white/80 dark:bg-zinc-900/80 backdrop-blur-sm">
            <CardContent className="flex flex-col items-center justify-center py-16">
              <Bot className="h-16 w-16 text-muted-foreground mb-4" />
              <h3 className="text-xl font-semibold mb-2">No agents yet</h3>
              <p className="text-muted-foreground text-center mb-6 max-w-md">
                Create your first AI agent to automate meeting participation and get real-time insights.
              </p>
              <Link href="/agents/create">
                <Button size="lg">
                  <Plus className="h-5 w-5 mr-2" />
                  Create Your First Agent
                </Button>
              </Link>
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
            {agents.map((agent) => (
              <Card key={agent.id} className="group hover:shadow-lg transition-all duration-300 hover:scale-[1.01] border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 overflow-hidden">
                <CardContent className="p-6">
                  {/* Header */}
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-center gap-3 flex-1 min-w-0">
                      <div className="relative">
                        <div className="p-2 bg-gray-100 dark:bg-gray-800 rounded-lg">
                          <Bot className="h-5 w-5 text-gray-600 dark:text-gray-400" />
                        </div>
                        <div className={`absolute -bottom-1 -right-1 w-3 h-3 rounded-full border-2 border-white dark:border-gray-900 ${
                          agent.status === 'running' ? 'bg-green-500' :
                          agent.status === 'error' ? 'bg-red-500' :
                          agent.status === 'starting' || agent.status === 'stopping' ? 'bg-yellow-500' :
                          'bg-gray-400'
                        }`} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-white truncate">{agent.config.name}</h3>
                        <div className="flex items-center gap-1 mt-1 text-gray-600 dark:text-gray-400">
                          <Globe className="h-4 w-4 flex-shrink-0" />
                          <span className="text-sm truncate">{new URL(agent.config.meeting_url).hostname}</span>
                        </div>
                      </div>
                    </div>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="sm" className="hover:bg-gray-100 dark:hover:bg-gray-800">
                          <MoreVertical className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <Link href={`/agents/${agent.id}`}>
                          <DropdownMenuItem>
                            <Eye className="h-4 w-4 mr-2" />
                            View Details
                          </DropdownMenuItem>
                        </Link>
                        <DropdownMenuItem onClick={() => handleDeleteAgent(agent.id)} disabled={agent.status === 'running'}>
                          <Trash2 className="h-4 w-4 mr-2" />
                          Delete Agent
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>

                  {/* Status */}
                  <div className="flex items-center gap-2 mb-4">
                    <div className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                      agent.status === 'running' ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300' :
                      agent.status === 'error' ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300' :
                      agent.status === 'starting' ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300' :
                      agent.status === 'stopping' ? 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300' :
                      'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300'
                    }`}>
                      {getStatusIcon(agent.status)}
                      <span className="ml-1 capitalize">
                        {agent.status === 'starting' && agent.config.auto_join ? 'Auto-starting...' : agent.status}
                      </span>
                    </div>
                    {agent.started_at && agent.status === 'running' && (
                      <div className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300">
                        <Clock className="h-3 w-3 mr-1" />
                        {formatUptime(agent.started_at)}
                      </div>
                    )}
                    {agent.status === 'created' && agent.config.auto_join && (
                      <div className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300">
                        <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                        Auto-starting...
                      </div>
                    )}
                  </div>

                  {/* AI Configuration */}
                  <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4">
                    <div className="flex items-center gap-2 mb-3">
                      <Zap className="h-4 w-4 text-gray-600 dark:text-gray-400" />
                      <span className="text-sm font-semibold text-gray-900 dark:text-white">AI Intelligence</span>
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <p className="text-xs text-gray-600 dark:text-gray-400 uppercase tracking-wide">Provider</p>
                        <p className="text-sm font-medium text-gray-900 dark:text-white capitalize">
                          {agent.config.llm_provider === 'google' ? 'Gemini' :
                           agent.config.llm_provider === 'openai' ? 'OpenAI' :
                           agent.config.llm_provider === 'anthropic' ? 'Claude' :
                           agent.config.llm_provider}
                        </p>
                      </div>
                      <div>
                        <p className="text-xs text-gray-600 dark:text-gray-400 uppercase tracking-wide">Model</p>
                        <p className="text-sm font-medium text-gray-900 dark:text-white truncate" title={agent.config.llm_model}>
                          {agent.config.llm_model}
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Voice & Language */}
                  <div className="grid grid-cols-2 gap-4">
                    <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-3">
                      <div className="flex items-center gap-2 mb-2">
                        <Mic className="h-4 w-4 text-gray-600 dark:text-gray-400" />
                        <span className="text-xs font-semibold text-gray-900 dark:text-white uppercase tracking-wide">Voice</span>
                      </div>
                      <p className="text-sm font-medium text-gray-900 dark:text-white capitalize">{agent.config.stt_provider}</p>
                      <p className="text-sm font-medium text-gray-900 dark:text-white capitalize">{agent.config.tts_provider}</p>
                    </div>

                    <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-3">
                      <div className="flex items-center gap-2 mb-2">
                        <MessageSquare className="h-4 w-4 text-gray-600 dark:text-gray-400" />
                        <span className="text-xs font-semibold text-gray-900 dark:text-white uppercase tracking-wide">Settings</span>
                      </div>
                      <p className="text-sm font-medium text-gray-900 dark:text-white uppercase">{agent.config.language}</p>
                      <div className="flex items-center gap-1 mt-1">
                        <div className={`w-2 h-2 rounded-full ${agent.config.name_trigger ? 'bg-green-500' : 'bg-gray-400'}`}></div>
                        <span className="text-xs text-gray-600 dark:text-gray-400">
                          {agent.config.name_trigger ? 'Name trigger' : 'No trigger'}
                        </span>
                      </div>
                    </div>
                  </div>

                  {/* Timestamps */}
                  <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400 pt-3 border-t border-gray-200 dark:border-gray-700">
                    <div className="flex items-center gap-1">
                      <Calendar className="h-3 w-3" />
                      <span>Created {new Date(agent.created_at).toLocaleDateString()}</span>
                    </div>
                    {agent.started_at && (
                      <div className="flex items-center gap-1">
                        <Activity className="h-3 w-3" />
                        <span>Active {new Date(agent.started_at).toLocaleDateString()}</span>
                      </div>
                    )}
                  </div>

                  {/* Actions */}
                  <div className="flex gap-2 pt-4">
                    {agent.status === 'running' ? (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleStopAgent(agent.id)}
                        className="flex-1 border-red-200 text-red-700 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950/20"
                      >
                        <Square className="h-4 w-4 mr-2" />
                        Stop
                      </Button>
                    ) : agent.status === 'stopped' || agent.status === 'error' ? (
                      <Button
                        size="sm"
                        onClick={() => handleStartAgent(agent.id)}
                        className="flex-1"
                      >
                        <Play className="h-4 w-4 mr-2" />
                        Start
                      </Button>
                    ) : (
                      <Button
                        disabled
                        size="sm"
                        className="flex-1"
                      >
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        {agent.status === 'starting' ? 'Starting...' : 'Stopping...'}
                      </Button>
                    )}
                    <Link href={`/agents/${agent.id}`}>
                      <Button variant="outline" size="sm" className="flex-1">
                        <Eye className="h-4 w-4 mr-2" />
                        Details
                      </Button>
                    </Link>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}

      </div>
    </Layout>
  );
}

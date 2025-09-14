/**
 * Professional create agent page with preview pane.
 */

'use client';

import { useState, useCallback, useRef, useEffect } from 'react';
import { PromptConfiguration } from './components/PromptConfiguration';
import { useRouter } from 'next/navigation';
import { Resolver, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import {
  ArrowLeft,
  Bot,
  Brain,
  Globe,
  MessageSquare,
  Mic,
  Volume2,
  Settings,
  Users,
  Target,
  Loader2,
  Save,
  Play,
  PanelLeftClose,
  PanelLeftOpen,
  Ear,
  Headphones,
  VolumeX,
  Eye,
  ChevronDown,
  ChevronRight
} from 'lucide-react';

import { Layout } from '@/components/layout/layout';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
import { Badge } from '@/components/ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import Image from 'next/image';
import { useAgentStore, useUIStore } from '@/lib/store';
import { agentsApi } from '@/lib/api';

const agentSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  meeting_url: z.string().url('Must be a valid URL'),
  conversation_mode: z.enum(['conversational', 'analyst']).default('conversational'),
  llm_provider: z.enum(['openai', 'anthropic', 'google', 'ollama']),
  llm_model: z.string().min(1, 'Model is required'),
  tts_provider: z.enum(['kokoro', 'elevenlabs', 'deepgram']),
  stt_provider: z.enum(['whisper', 'deepgram']),
  language: z.string().min(1, 'Language is required'),
  custom_prompt: z.string().optional(),
  personality_prompt: z.string().optional(),

  name_trigger: z.boolean(),
  auto_join: z.boolean(),
  // Advanced parameters
  utterance_tail_seconds: z.number().min(0.1).max(5.0).optional(),
  no_speech_event_delay: z.number().min(0.1).max(2.0).optional(),
  max_stt_tasks: z.number().min(1).max(20).optional(),
  window_queue_size: z.number().min(10).max(1000).optional(),
  stt_args: z.record(z.string(), z.any()).optional(),
  tts_args: z.record(z.string(), z.any()).optional(),
  vad_args: z.record(z.string(), z.any()).optional(),
  env_vars: z.record(z.string(), z.string()),
});

type AgentFormData = z.infer<typeof agentSchema>;

const PROVIDER_OPTIONS = [
  {
    value: 'openai',
    label: 'OpenAI',
    lightLogo: '/logos/openai (1).png',
    darkLogo: '/logos/openai.png',
    description: 'Industry-leading AI models with GPT-4',
    caption: 'Best for complex reasoning and creative tasks',
    color: 'bg-slate-50 border-slate-200 hover:bg-slate-100 dark:bg-slate-900/50 dark:border-slate-700 dark:hover:bg-slate-800/50',
    selectedColor: 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900 dark:border-slate-100',
    models: [
      { value: 'gpt-4o', label: 'GPT-4o', description: 'Most capable and balanced' },
      { value: 'gpt-4', label: 'GPT-4', description: 'Maximum intelligence' },
      { value: 'gpt-3.5-turbo', label: 'GPT-3.5 Turbo', description: 'Fast and cost-effective' },
    ]
  },
  {
    value: 'anthropic',
    label: 'Anthropic',
    lightLogo: '/logos/anthropic-logo.png',
    darkLogo: '/logos/anthropic-logo.png',
    description: 'Safety-focused AI with Claude models',
    caption: 'Best for ethical AI and safety-critical tasks',
    color: 'bg-slate-50 border-slate-200 hover:bg-slate-100 dark:bg-slate-900/50 dark:border-slate-700 dark:hover:bg-slate-800/50',
    selectedColor: 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900 dark:border-slate-100',
    models: [
      { value: 'claude-3-5-sonnet-20241022', label: 'Claude 3.5 Sonnet', description: 'Most intelligent and helpful' },
      { value: 'claude-3-haiku-20240307', label: 'Claude 3 Haiku', description: 'Fast and efficient' },
    ]
  },
  {
    value: 'google',
    label: 'Google',
    lightLogo: '/logos/google-logo.png',
    darkLogo: '/logos/google-logo.png',
    description: 'Google\'s Gemini AI with multimodal capabilities',
    caption: 'Best for multimodal tasks and Google ecosystem',
    color: 'bg-slate-50 border-slate-200 hover:bg-slate-100 dark:bg-slate-900/50 dark:border-slate-700 dark:hover:bg-slate-800/50',
    selectedColor: 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900 dark:border-slate-100',
    models: [
      { value: 'gemini-2.5-flash', label: 'Gemini 2.5 Flash', description: 'Latest and most capable model' },
      { value: 'gemini-2.5-flash-lite', label: 'Gemini 2.5 Flash Lite', description: 'Lighter and faster version' },
      { value: 'gemini-2.0-flash', label: 'Gemini 2.0 Flash', description: 'Advanced multimodal capabilities' },
      { value: 'gemini-1.5-pro', label: 'Gemini 1.5 Pro', description: 'Advanced multimodal understanding' },
      { value: 'gemini-1.5-flash', label: 'Gemini 1.5 Flash', description: 'Fast responses with good quality' },
    ]
  },
  {
    value: 'ollama',
    label: 'Ollama',
    lightLogo: '/logos/ollama.png',
    darkLogo: '/logos/ollama (1).png',
    description: 'Run AI models locally on your machine',
    caption: 'Best for privacy and offline capabilities',
    color: 'bg-slate-50 border-slate-200 hover:bg-slate-100 dark:bg-slate-900/50 dark:border-slate-700 dark:hover:bg-slate-800/50',
    selectedColor: 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900 dark:border-slate-100',
    models: [
      { value: 'llama3.2', label: 'Llama 3.2', description: 'Modern and capable open-source model' },
      { value: 'mistral', label: 'Mistral', description: 'Efficient and fast inference' },
    ]
  }
];

const TTS_OPTIONS = [
  {
    value: 'kokoro',
    label: 'Kokoro',
    icon: Volume2,
    description: 'Local text-to-speech engine',
    caption: 'Free, fast, and private - runs on your machine',
    color: 'bg-slate-50 border-slate-200 hover:bg-slate-100 dark:bg-slate-900/50 dark:border-slate-700 dark:hover:bg-slate-800/50',
    selectedColor: 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900 dark:border-slate-100'
  },
  {
    value: 'elevenlabs',
    label: 'ElevenLabs',
    icon: Mic,
    description: 'Premium voice synthesis',
    caption: 'High-quality, natural-sounding voices with emotion',
    color: 'bg-slate-50 border-slate-200 hover:bg-slate-100 dark:bg-slate-900/50 dark:border-slate-700 dark:hover:bg-slate-800/50',
    selectedColor: 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900 dark:border-slate-100'
  },
  {
    value: 'deepgram',
    label: 'Deepgram',
    icon: VolumeX,
    description: 'Neural voice generation',
    caption: 'Advanced AI voices with realistic intonation',
    color: 'bg-slate-50 border-slate-200 hover:bg-slate-100 dark:bg-slate-900/50 dark:border-slate-700 dark:hover:bg-slate-800/50',
    selectedColor: 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900 dark:border-slate-100'
  }
];

const STT_OPTIONS = [
  {
    value: 'whisper',
    label: 'Whisper',
    icon: Ear,
    description: 'OpenAI\'s speech recognition',
    caption: 'Highly accurate, runs locally for maximum privacy',
    color: 'bg-slate-50 border-slate-200 hover:bg-slate-100 dark:bg-slate-900/50 dark:border-slate-700 dark:hover:bg-slate-800/50',
    selectedColor: 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900 dark:border-slate-100'
  },
  {
    value: 'deepgram',
    label: 'Deepgram',
    icon: Headphones,
    description: 'Cloud speech processing',
    caption: 'Fast, accurate transcription with advanced features',
    color: 'bg-slate-50 border-slate-200 hover:bg-slate-100 dark:bg-slate-900/50 dark:border-slate-700 dark:hover:bg-slate-800/50',
    selectedColor: 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900 dark:border-slate-100'
  }
];

const LANGUAGE_OPTIONS = [
  { value: 'en', label: 'English', flag: '🇺🇸', description: 'American English - most compatible' },
  { value: 'de', label: 'German', flag: '🇩🇪', description: 'Deutsch - European German' },
  { value: 'fr', label: 'French', flag: '🇫🇷', description: 'Français - European French' },
  { value: 'es', label: 'Spanish', flag: '🇪🇸', description: 'Español - European Spanish' },
  { value: 'it', label: 'Italian', flag: '🇮🇹', description: 'Italiano - Standard Italian' },
];

const MEETING_STYLE_OPTIONS = [
  {
    value: 'mpc',
    label: 'Group Meetings',
    icon: Users,
    description: 'Multi-participant conversations',
    caption: 'Optimized for team meetings with multiple speakers',
    color: 'bg-slate-50 border-slate-200 hover:bg-slate-100 dark:bg-slate-900/50 dark:border-slate-700 dark:hover:bg-slate-800/50',
    selectedColor: 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900 dark:border-slate-100'
  },
  {
    value: 'dyadic',
    label: 'One-on-One',
    icon: Target,
    description: 'Direct conversations',
    caption: 'Focused on personal, direct interactions',
    color: 'bg-slate-50 border-slate-200 hover:bg-slate-100 dark:bg-slate-900/50 dark:border-slate-700 dark:hover:bg-slate-800/50',
    selectedColor: 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900 dark:border-slate-100'
  }
];

const CONVERSATION_MODE_OPTIONS = [
  {
    value: 'conversational',
    label: 'Conversational',
    icon: MessageSquare,
    description: 'Active participation in meetings',
    caption: 'Agent listens, responds, and speaks during meetings',
    color: 'bg-slate-50 border-slate-200 hover:bg-slate-100 dark:bg-slate-900/50 dark:border-slate-700 dark:hover:bg-slate-800/50',
    selectedColor: 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900 dark:border-slate-100',
    workflow: [
      { step: 'Agent joins meeting', color: 'bg-blue-500' },
      { step: 'Listens to conversation', color: 'bg-green-500' },
      { step: 'Processes and responds', color: 'bg-purple-500' },
      { step: 'Speaks responses', color: 'bg-orange-500' }
    ]
  },
  {
    value: 'analyst',
    label: 'Analyst Mode',
    icon: Eye,
    description: 'Silent analysis and note-taking',
    caption: 'Agent transcribes and analyzes without speaking',
    color: 'bg-slate-50 border-slate-200 hover:bg-slate-100 dark:bg-slate-900/50 dark:border-slate-700 dark:hover:bg-slate-800/50',
    selectedColor: 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900 dark:border-slate-100',
    workflow: [
      { step: 'Agent joins meeting', color: 'bg-blue-500' },
      { step: 'Transcribes everything', color: 'bg-green-500' },
      { step: 'Analyzes content', color: 'bg-purple-500' },
      { step: 'Generates insights', color: 'bg-orange-500' }
    ]
  }
];

// Utility function to get the appropriate logo based on current theme and selection state
const getProviderLogo = (provider: typeof PROVIDER_OPTIONS[0], isDarkMode: boolean, isSelected: boolean = false) => {
  // When selected, use the opposite logo because selected items have inverted backgrounds
  const useOppositeLogo = isSelected;
  const effectiveDarkMode = isDarkMode !== useOppositeLogo; // XOR operation
  return effectiveDarkMode ? provider.darkLogo : provider.lightLogo;
};

export default function CreateAgentPage() {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [envVarKey, setEnvVarKey] = useState('');
  const [showPreview, setShowPreview] = useState(true);
  const [previewKey, setPreviewKey] = useState(0); // For triggering preview animations
  const [envVarValue, setEnvVarValue] = useState('');
  const [focusedGroup, setFocusedGroup] = useState<string | null>(null);
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});
  const [isDarkMode, setIsDarkMode] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);

  // Initialize and monitor theme changes
  useEffect(() => {
    const checkTheme = () => {
      const isDark = document.documentElement.classList.contains('dark');
      setIsDarkMode(isDark);
    };

    // Check initial theme
    checkTheme();

    // Create a mutation observer to watch for theme changes
    const observer = new MutationObserver(checkTheme);
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class']
    });

    return () => observer.disconnect();
  }, []);

  // Refs for accessibility and keyboard navigation
  const providerGroupRef = useRef<HTMLDivElement>(null);
  const modelGroupRef = useRef<HTMLDivElement>(null);
  const ttsGroupRef = useRef<HTMLDivElement>(null);
  const sttGroupRef = useRef<HTMLDivElement>(null);

  const { updateAgent } = useAgentStore();
  const { addNotification } = useUIStore();

  // URL validation helper
  const validateMeetingUrl = (url: string) => {
    if (!url) return true; // Optional field
    const meetingUrlPattern = /^https?:\/\/(meet\.google\.com|zoom\.us|teams\.microsoft\.com|teams\.live\.com)\/[^\s]+$/i;
    return meetingUrlPattern.test(url);
  };

  // Extract domain for preview
  const getMeetingDomain = (url: string) => {
    if (!url) return null;
    try {
      const urlObj = new URL(url);
      return urlObj.hostname;
    } catch {
      return null;
    }
  };

  // Trigger preview animation when form changes
  const triggerPreviewUpdate = useCallback(() => {
    setPreviewKey(prev => prev + 1);
  }, []);

  // Keyboard navigation handlers for radio groups
  const handleRadioKeyDown = useCallback((
    event: React.KeyboardEvent,
    options: { value: string }[],
    currentValue: string,
    setValue: (value: string) => void
  ) => {
    const currentIndex = options.findIndex(option => option.value === currentValue);

    switch (event.key) {
      case 'ArrowLeft':
      case 'ArrowUp':
        event.preventDefault();
        const prevIndex = currentIndex > 0 ? currentIndex - 1 : options.length - 1;
        setValue(options[prevIndex].value);
        triggerPreviewUpdate();
        break;
      case 'ArrowRight':
      case 'ArrowDown':
        event.preventDefault();
        const nextIndex = currentIndex < options.length - 1 ? currentIndex + 1 : 0;
        setValue(options[nextIndex].value);
        triggerPreviewUpdate();
        break;
      case 'Home':
        event.preventDefault();
        setValue(options[0].value);
        triggerPreviewUpdate();
        break;
      case 'End':
        event.preventDefault();
        setValue(options[options.length - 1].value);
        triggerPreviewUpdate();
        break;
    }
  }, [triggerPreviewUpdate]);

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    getValues,
    formState: { errors },
  } = useForm<AgentFormData>({
    resolver: zodResolver(agentSchema) as Resolver<AgentFormData>,
    mode: 'onChange',
    defaultValues: {
      name: '',
      meeting_url: '',
      conversation_mode: 'conversational',
      llm_provider: 'google',
      llm_model: 'gemini-2.5-flash-lite',
      tts_provider: 'kokoro',
      stt_provider: 'whisper',
      language: 'en',
      custom_prompt: '',
      personality_prompt: '',

      name_trigger: false,
      auto_join: true,
      utterance_tail_seconds: 1.0,
      no_speech_event_delay: 0.5,
      max_stt_tasks: 5,
      window_queue_size: 10,
      stt_args: {},
      tts_args: {},
      vad_args: {},
      env_vars: {},
    },
  });

  // Form validation effects
  useEffect(() => {
    const subscription = watch((value, { name }) => {
      if (name === 'meeting_url' && value.meeting_url) {
        const isValid = validateMeetingUrl(value.meeting_url);
        setFormErrors(prev => ({
          ...prev,
          meeting_url: isValid ? '' : 'Please enter a valid Google Meet, Zoom, or Teams URL'
        }));
      }
      triggerPreviewUpdate();
    });
    return () => subscription.unsubscribe();
  }, [watch, triggerPreviewUpdate]);

  const watchedProvider = watch('llm_provider');
  const watchedTTS = watch('tts_provider');
  const watchedSTT = watch('stt_provider');
  const watchedLanguage = watch('language');
  const envVars = watch('env_vars');
  const watchedConversationMode = watch('conversation_mode');
  const handleProviderChange = (provider: string) => {
    setValue('llm_provider', provider as 'openai' | 'anthropic' | 'google' | 'ollama');

    // Set default model for each provider
    const defaultModels = {
      'openai': 'gpt-4o',
      'anthropic': 'claude-3-5-sonnet-20241022',
      'google': 'gemini-2.5-flash-lite',
      'ollama': 'llama3.2'
    };

    const defaultModel = defaultModels[provider as keyof typeof defaultModels];
    if (defaultModel) {
      setValue('llm_model', defaultModel);
    }
  };

  const handleModelChange = (model: string) => {
    setValue('llm_model', model);
  };

  const addEnvVar = () => {
    if (envVarKey && envVarValue) {
      const currentEnvVars = getValues('env_vars');
      setValue('env_vars', {
        ...currentEnvVars,
        [envVarKey]: envVarValue,
      });
      setEnvVarKey('');
      setEnvVarValue('');
    }
  };

  const removeEnvVar = (key: string) => {
    const currentEnvVars = getValues('env_vars');
    const newEnvVars = { ...currentEnvVars };
    delete newEnvVars[key];
    setValue('env_vars', newEnvVars);
  };

  const handleFormSubmit = async (data: AgentFormData) => {
    setIsSubmitting(true);
    try {
      const response = await agentsApi.create(data);
      updateAgent(response.data);

      // Provide different messages based on auto_join setting
      if (data.auto_join) {
        addNotification({
          type: 'success',
          title: 'Agent created and starting',
          message: `Agent "${response.data.config.name}" has been created and is auto-starting.`,
        });
      } else {
        addNotification({
          type: 'success',
          title: 'Agent created successfully',
          message: `Agent "${response.data.config.name}" has been created.`,
        });
      }

      router.push('/agents');
    } catch (error) {
      console.error('Failed to create agent:', error);
      addNotification({
        type: 'error',
        title: 'Failed to create agent',
        message: 'Could not create agent. Please check your configuration.',
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const selectedProvider = PROVIDER_OPTIONS.find(p => p.value === watchedProvider);
  const selectedTTS = TTS_OPTIONS.find(t => t.value === watchedTTS);
  const selectedSTT = STT_OPTIONS.find(s => s.value === watchedSTT);
  const selectedLanguage = LANGUAGE_OPTIONS.find(l => l.value === watchedLanguage);
  const selectedConversationMode = CONVERSATION_MODE_OPTIONS.find(c => c.value === watchedConversationMode);
  return (
    <Layout title="Create Agent" subtitle="Configure your AI meeting assistant">
      <div className="max-w-6xl mx-auto">
        {/* Header with Preview Toggle */}
        <div className="flex items-center justify-between p-4 bg-white dark:bg-zinc-900 backdrop-blur-lg rounded-xl border border-slate-200 dark:border-zinc-800 shadow-sm mb-6">
          <div className="flex items-center gap-4">
            <Button
              variant="outline"
              size="sm"
              onClick={() => router.back()}
              className="gap-2"
            >
              <ArrowLeft className="h-4 w-4" />
              Back to Agents
            </Button>
            <div>
              <h1 className="text-2xl font-bold tracking-tight">Create New Agent</h1>
              <p className="text-sm text-muted-foreground">
                Configure your AI meeting assistant
              </p>
            </div>
          </div>

          {/* Preview Toggle */}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowPreview(!showPreview)}
            className="gap-2"
          >
            {showPreview ? (
              <>
                <PanelLeftClose className="h-4 w-4" />
                Hide Preview
              </>
            ) : (
              <>
                <PanelLeftOpen className="h-4 w-4" />
                Show Preview
              </>
            )}
          </Button>
        </div>

        <div className="max-w-7xl mx-auto">
          {/* Responsive Grid Layout */}
          <div className="grid grid-cols-1 xl:grid-cols-[1fr_360px] gap-8 lg:gap-12">
            {/* Main Form Column */}
            <div className="space-y-8">
              <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-8">
                {/* Form Container */}
                <div className="bg-white dark:bg-zinc-900/50 border border-slate-200 dark:border-zinc-800 rounded-xl p-6 lg:p-8 space-y-10">

                    {/* Agent Basics Section */}
                    <div className="space-y-6">
                      <div className="space-y-2">
                        <div className="flex items-center gap-3">
                          <div className="p-2 bg-slate-100 dark:bg-zinc-800 rounded-lg">
                            <Bot className="h-5 w-5 text-slate-600 dark:text-zinc-400" />
                          </div>
                          <div>
                            <h2 id="agent-basics-heading" className="text-2xl font-semibold">Agent Basics</h2>
                            <p className="text-sm text-slate-600 dark:text-zinc-400 mt-1">
                              Set up your agent&apos;s identity and meeting details
                            </p>
                          </div>
                        </div>
                      </div>

                      {/* Primary Fields */}
                      <div className="space-y-6">
                        <div className="grid gap-6 grid-cols-1">
                          {/* Agent Name Field */}
                          <div className="space-y-3">
                            <div className="space-y-2">
                              <Label htmlFor="name" className="text-base font-semibold flex items-center gap-2 text-slate-900 dark:text-slate-100">
                                <Bot className="h-4 w-4 text-blue-600" />
                                Agent Name
                                <span className="text-red-500 ml-1" aria-label="required">*</span>
                              </Label>
                              <p className="text-sm text-slate-600 dark:text-zinc-400">Give your AI assistant a memorable name</p>
                            </div>
                            <Input
                              id="name"
                              {...register('name')}
                              placeholder="e.g., Alex Meeting Assistant"
                              className="h-12 text-base border-2 border-blue-200 dark:border-blue-700 focus:border-blue-500 dark:focus:border-blue-500 bg-white dark:bg-slate-900/50 transition-colors rounded-lg"
                              aria-describedby="name-description"
                              aria-invalid={!!errors.name}
                            />
                            <div id="name-description" className="sr-only">Enter a descriptive name for your AI agent</div>
                            {errors.name && (
                              <p className="text-sm text-red-600" role="alert" aria-live="polite">
                                {errors.name.message}
                              </p>
                            )}
                          </div>

                          {/* Meeting URL Field */}
                          <div className="space-y-3">
                            <div className="space-y-2">
                              <Label htmlFor="meeting_url" className="text-base font-semibold flex items-center gap-2 text-slate-900 dark:text-slate-100">
                                <Globe className="h-4 w-4 text-blue-600" />
                                Meeting URL
                              </Label>
                              <p className="text-sm text-slate-600 dark:text-zinc-400">Paste your Google Meet, Zoom, or Teams meeting link</p>
                            </div>
                            <Input
                              id="meeting_url"
                              {...register('meeting_url')}
                              placeholder="https://meet.google.com/abc-defg-hij"
                              className="h-12 text-base border-2 border-blue-200 dark:border-blue-700 focus:border-blue-500 dark:focus:border-blue-500 bg-white dark:bg-slate-900/50 transition-colors rounded-lg"
                              aria-describedby="meeting-url-description"
                              aria-invalid={!!formErrors.meeting_url}
                            />
                            <div id="meeting-url-description" className="sr-only">
                              Enter the full URL for your video meeting. Supported platforms: Google Meet, Zoom, Microsoft Teams
                            </div>
                            {formErrors.meeting_url && (
                              <p className="text-sm text-red-600" role="alert" aria-live="polite">
                                {formErrors.meeting_url}
                              </p>
                            )}
                          </div>
                        </div>
                      </div>

                      <div className="flex items-start gap-3 p-4 bg-slate-50 dark:bg-zinc-900/50 rounded-lg border border-slate-200 dark:border-zinc-700">
                        <Checkbox
                          id="auto_join"
                          {...register('auto_join')}
                          defaultChecked
                          className="mt-0.5"
                          aria-describedby="auto-join-description"
                        />
                        <div className="space-y-1">
                          <Label htmlFor="auto_join" className="text-sm font-medium cursor-pointer">
                            Auto-join meetings
                            {watch('auto_join') && (
                              <span className="ml-2 inline-flex items-center px-2 py-1 text-xs bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300 rounded-full">
                                <Play className="h-3 w-3 mr-1" />
                                Auto-start enabled
                              </span>
                            )}
                          </Label>
                          <p id="auto-join-description" className="text-xs text-slate-600 dark:text-zinc-400">
                            Agent will automatically join when the meeting starts, eliminating manual setup
                            {watch('auto_join') && (
                              <span className="block mt-1 text-green-600 dark:text-green-400 font-medium">
                                ✓ Agent will start automatically after creation
                              </span>
                            )}
                          </p>
                        </div>
                      </div>
                    </div>

                    {/* Conversation Mode Section */}
                    <div className="space-y-6">
                      <div className="space-y-2">
                        <div className="flex items-center gap-3">
                          <div className="p-2 bg-slate-100 dark:bg-zinc-800 rounded-lg">
                            <MessageSquare className="h-5 w-5 text-slate-600 dark:text-zinc-400" />
                          </div>
                          <div>
                            <h2 id="conversation-mode-heading" className="text-2xl font-semibold">Conversation Mode</h2>
                            <p className="text-sm text-slate-600 dark:text-zinc-400 mt-1">
                              Choose how your agent behaves in meetings
                            </p>
                          </div>
                        </div>
                      </div>

                      <div className="space-y-4">
                        <div className="space-y-2">
                          <Label className="text-sm font-medium">Agent Behavior <span className="text-red-500" aria-label="required">*</span></Label>
                          <p className="text-xs text-slate-500 dark:text-zinc-500">Select the primary mode for your agent</p>
                        </div>

                        <div
                          role="radiogroup"
                          aria-labelledby="conversation-mode-label"
                          aria-describedby="conversation-mode-description"
                          className="grid gap-4 grid-cols-1 lg:grid-cols-2"
                        >
                          {CONVERSATION_MODE_OPTIONS.map((mode, index) => (
                            <button
                              key={mode.value}
                              role="radio"
                              aria-checked={watch('conversation_mode') === mode.value}
                              aria-describedby={`conversation-mode-${mode.value}-description`}
                              tabIndex={watch('conversation_mode') === mode.value ? 0 : -1}
                              type="button"
                              onClick={() => setValue('conversation_mode', mode.value as 'conversational' | 'analyst')}
                              onKeyDown={(e) => handleRadioKeyDown(e, CONVERSATION_MODE_OPTIONS, watch('conversation_mode'), (value) => setValue('conversation_mode', value as 'conversational' | 'analyst'))}
                              className={`p-6 rounded-lg border transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
                                watch('conversation_mode') === mode.value
                                  ? `${mode.selectedColor} ring-2 ring-blue-500 ring-offset-2`
                                  : `${mode.color} hover:shadow-lg`
                              }`}
                            >
                              <div className="text-center space-y-4">
                                <div className="w-12 h-12 mx-auto flex items-center justify-center rounded-full bg-slate-100 dark:bg-zinc-700">
                                  <mode.icon className="h-6 w-6 text-slate-600 dark:text-zinc-400" />
                                </div>
                                <div className="space-y-2">
                                  <div className="text-lg font-semibold">{mode.label}</div>
                                  <div className="text-sm text-slate-600 dark:text-zinc-400">{mode.description}</div>
                                  <div className="text-xs opacity-75">{mode.caption}</div>
                                </div>
                              </div>
                              <div id={`conversation-mode-${mode.value}-description`} className="sr-only">
                                {mode.description}. {mode.caption}
                              </div>
                            </button>
                          ))}
                        </div>
                        <div id="conversation-mode-description" className="sr-only">
                          Choose how your agent behaves in meetings. Conversational mode allows active participation, Analyst mode provides silent analysis and note-taking.
                        </div>
                      </div>

                      {/* Mode-specific information */}
                      {watch('conversation_mode') === 'analyst' && (
                        <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-700">
                          <div className="flex items-start gap-3">
                            <div className="w-5 h-5 bg-blue-500 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5">
                              <Eye className="h-3 w-3 text-white" />
                            </div>
                            <div className="space-y-2">
                              <div className="text-sm font-medium text-blue-900 dark:text-blue-100">
                                Analyst Mode Features
                              </div>
                              <div className="text-xs text-blue-700 dark:text-blue-300 space-y-1">
                                <div>✓ Comprehensive meeting transcription</div>
                                <div>✓ Automated summary generation</div>
                                <div>✓ Key points and action items extraction</div>
                                <div>✓ Topic analysis and sentiment detection</div>
                                <div>✓ Structured analysis reports via API</div>
                              </div>
                            </div>
                          </div>
                        </div>
                      )}

                      {watch('conversation_mode') === 'conversational' && (
                        <div className="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-700">
                          <div className="flex items-start gap-3">
                            <div className="w-5 h-5 bg-green-500 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5">
                              <MessageSquare className="h-3 w-3 text-white" />
                            </div>
                            <div className="space-y-2">
                              <div className="text-sm font-medium text-green-900 dark:text-green-100">
                                Conversational Mode Features
                              </div>
                              <div className="text-xs text-green-700 dark:text-green-300 space-y-1">
                                <div>✓ Active participation in meetings</div>
                                <div>✓ Real-time responses and conversation</div>
                                <div>✓ Voice synthesis and natural speech</div>
                                <div>✓ Contextual understanding and memory</div>
                                <div>✓ Customizable response prompts</div>
                              </div>
                            </div>
                          </div>
                        </div>
                      )}
                    </div>

                    {/* AI Configuration Section */}
                    <div className="space-y-8">
                      <div className="space-y-2">
                        <div className="flex items-center gap-3">
                          <div className="p-2 bg-slate-100 dark:bg-zinc-800 rounded-lg">
                            <Brain className="h-5 w-5 text-slate-600 dark:text-zinc-400" />
                          </div>
                          <div>
                            <h2 id="ai-config-heading" className="text-2xl font-semibold">AI Configuration</h2>
                            <p className="text-sm text-slate-600 dark:text-zinc-400 mt-1">
                              Choose your AI provider and speech capabilities
                            </p>
                          </div>
                        </div>
                      </div>

                      {/* Provider Selection */}
                      <div className="space-y-4">
                        <div className="space-y-2">
                          <Label className="text-sm font-medium">AI Provider <span className="text-red-500" aria-label="required">*</span></Label>
                          <p className="text-xs text-slate-500 dark:text-zinc-500">Select the AI service for your agent&apos;s intelligence</p>
                        </div>

                        <div
                          ref={providerGroupRef}
                          role="radiogroup"
                          aria-labelledby="provider-label"
                          aria-describedby="provider-description"
                          className="grid gap-3 grid-cols-2"
                          onFocus={() => setFocusedGroup('provider')}
                          onBlur={() => setFocusedGroup(null)}
                        >
                          {PROVIDER_OPTIONS.map((provider, index) => (
                            <button
                              key={provider.value}
                              role="radio"
                              aria-checked={watchedProvider === provider.value}
                              aria-describedby={`provider-${provider.value}-description`}
                              tabIndex={watchedProvider === provider.value ? 0 : -1}
                              type="button"
                              onClick={() => handleProviderChange(provider.value)}
                              onKeyDown={(e) => handleRadioKeyDown(e, PROVIDER_OPTIONS, watchedProvider, handleProviderChange)}
                              className={`p-4 rounded-lg border transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
                                watchedProvider === provider.value
                                  ? `${provider.selectedColor} ring-2 ring-blue-500 ring-offset-2`
                                  : `${provider.color} hover:shadow-md`
                              }`}
                            >
                              <div className="text-center space-y-2">
                                <div className="w-8 h-8 mx-auto flex items-center justify-center">
                                  <Image
                                    src={getProviderLogo(provider, isDarkMode, watchedProvider === provider.value)}
                                    alt=""
                                    width={32}
                                    height={32}
                                    className="object-contain"
                                  />
                                </div>
                                <div className="space-y-1">
                                  <div className="text-sm font-medium">{provider.label}</div>
                                  <div className="text-xs opacity-80">{provider.caption}</div>
                                </div>
                              </div>
                              <div id={`provider-${provider.value}-description`} className="sr-only">
                                {provider.description}. {provider.caption}
                              </div>
                            </button>
                          ))}
                        </div>
                        <div id="provider-description" className="sr-only">
                          Choose an AI provider for your agent. Use arrow keys to navigate between options, space or enter to select.
                        </div>
                      </div>

                      {/* Model Selection */}
                      {selectedProvider && (
                        <div className="space-y-4">
                          <div className="space-y-2">
                            <Label className="text-sm font-medium">AI Model <span className="text-red-500" aria-label="required">*</span></Label>
                            <p className="text-xs text-slate-500 dark:text-zinc-500">Choose the specific model for optimal performance</p>
                          </div>

                          <div
                            ref={modelGroupRef}
                            role="radiogroup"
                            aria-labelledby="model-label"
                            aria-describedby="model-description"
                            className="flex flex-wrap gap-2"
                            onFocus={() => setFocusedGroup('model')}
                            onBlur={() => setFocusedGroup(null)}
                          >
                            {selectedProvider.models.map((model, index) => (
                              <button
                                key={model.value}
                                role="radio"
                                aria-checked={watch('llm_model') === model.value}
                                aria-describedby={`model-${model.value}-description`}
                                tabIndex={watch('llm_model') === model.value ? 0 : -1}
                                type="button"
                                onClick={() => handleModelChange(model.value)}
                                onKeyDown={(e) => handleRadioKeyDown(e, selectedProvider.models, watch('llm_model'), handleModelChange)}
                                className={`px-4 py-2.5 rounded-lg border text-sm transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
                                  watch('llm_model') === model.value
                                    ? 'bg-slate-900 text-white border-slate-900 ring-2 ring-blue-500 ring-offset-2 dark:bg-slate-100 dark:text-slate-900'
                                    : 'bg-slate-50 border-slate-200 hover:bg-slate-100 hover:shadow-sm dark:bg-slate-900/50 dark:border-slate-700'
                                }`}
                              >
                                <div className="text-center">
                                  <div className="font-medium">{model.label}</div>
                                  <div className="text-xs opacity-75 mt-0.5">{model.description}</div>
                                </div>
                                <div id={`model-${model.value}-description`} className="sr-only">
                                  {model.description}
                                </div>
                              </button>
                            ))}
                          </div>
                          <div id="model-description" className="sr-only">
                            Choose an AI model for your agent. Use arrow keys to navigate between options, space or enter to select.
                          </div>
                        </div>
                      )}

                        {/* Voice & Language Configuration */}
                      <div className={`grid gap-6 ${watchedConversationMode === 'analyst' ? 'lg:grid-cols-1' : 'lg:grid-cols-2'}`}>
                        {/* Text-to-Speech - Only show if not in analyst mode */}
                        {watchedConversationMode !== 'analyst' && (
                          <div className="space-y-4">
                            <div className="space-y-2">
                              <Label className="text-sm font-medium flex items-center gap-2">
                                <Volume2 className="h-4 w-4" />
                                Text-to-Speech <span className="text-red-500" aria-label="required">*</span>
                              </Label>
                              <p className="text-xs text-slate-500 dark:text-zinc-500">Choose how your agent speaks</p>
                            </div>

                          <div
                            ref={ttsGroupRef}
                            role="radiogroup"
                            aria-labelledby="tts-label"
                            aria-describedby="tts-description"
                            className="space-y-3"
                            onFocus={() => setFocusedGroup('tts')}
                            onBlur={() => setFocusedGroup(null)}
                          >
                            {TTS_OPTIONS.map((option, index) => (
                              <button
                                key={option.value}
                                role="radio"
                                aria-checked={watchedTTS === option.value}
                                aria-describedby={`tts-${option.value}-description`}
                                tabIndex={watchedTTS === option.value ? 0 : -1}
                                type="button"
                                onClick={() => setValue('tts_provider', option.value as 'kokoro' | 'elevenlabs' | 'deepgram')}
                                onKeyDown={(e) => handleRadioKeyDown(e, TTS_OPTIONS, watchedTTS, (value) => setValue('tts_provider', value as 'kokoro' | 'elevenlabs' | 'deepgram'))}
                                className={`w-full p-3 rounded-lg border transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
                                  watchedTTS === option.value
                                    ? `${option.selectedColor} ring-2 ring-blue-500 ring-offset-2`
                                    : `${option.color} hover:shadow-sm`
                                }`}
                              >
                                <div className="flex items-start gap-3">
                                  <option.icon className="h-5 w-5 mt-0.5 text-slate-600 dark:text-zinc-400" />
                                  <div className="text-left">
                                    <div className="text-sm font-medium">{option.label}</div>
                                    <div className="text-xs opacity-75 mt-0.5">{option.caption}</div>
                                  </div>
                                </div>
                                <div id={`tts-${option.value}-description`} className="sr-only">
                                  {option.description}. {option.caption}
                                </div>
                              </button>
                            ))}
                          </div>
                          <div id="tts-description" className="sr-only">
                            Choose a text-to-speech provider for your agent. Use arrow keys to navigate between options, space or enter to select.
                          </div>
                        </div>
                        )}

                        {/* Language Selection - Dropdown */}
                    <div className="space-y-4">
                      <div className="space-y-2">
                        <Label className="text-sm font-medium flex items-center gap-2">
                          <MessageSquare className="h-4 w-4" />
                          Language
                        </Label>
                        <p className="text-xs text-slate-500 dark:text-zinc-500">Select the primary language for your agent</p>
                      </div>
                      <Select value={watchedLanguage} onValueChange={(value) => setValue('language', value)}>
                        <SelectTrigger className="h-11">
                          <SelectValue placeholder="Choose language" />
                        </SelectTrigger>
                        <SelectContent>
                          {LANGUAGE_OPTIONS.map((lang) => (
                            <SelectItem key={lang.value} value={lang.value}>
                              <div className="flex items-center gap-3">
                                <span className="text-lg">{lang.flag}</span>
                                <div>
                                  <div className="font-medium">{lang.label}</div>
                                  <div className="text-xs text-muted-foreground">{lang.description}</div>
                                </div>
                              </div>
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                        {/* Speech-to-Text */}
                        <div className="space-y-4">
                          <div className="space-y-2">
                            <Label className="text-sm font-medium flex items-center gap-2">
                              <Mic className="h-4 w-4" />
                              Speech-to-Text <span className="text-red-500" aria-label="required">*</span>
                            </Label>
                            <p className="text-xs text-slate-500 dark:text-zinc-500">Choose how your agent understands speech</p>
                          </div>

                          <div
                            ref={sttGroupRef}
                            role="radiogroup"
                            aria-labelledby="stt-label"
                            aria-describedby="stt-description"
                            className="grid gap-3 grid-cols-1 sm:grid-cols-2"
                            onFocus={() => setFocusedGroup('stt')}
                            onBlur={() => setFocusedGroup(null)}
                          >
                            {STT_OPTIONS.map((option, index) => (
                              <button
                                key={option.value}
                                role="radio"
                                aria-checked={watchedSTT === option.value}
                                aria-describedby={`stt-${option.value}-description`}
                                tabIndex={watchedSTT === option.value ? 0 : -1}
                                type="button"
                                onClick={() => setValue('stt_provider', option.value as 'whisper' | 'deepgram')}
                                onKeyDown={(e) => handleRadioKeyDown(e, STT_OPTIONS, watchedSTT, (value) => setValue('stt_provider', value as 'whisper' | 'deepgram'))}
                                className={`p-3 rounded-lg border transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
                                  watchedSTT === option.value
                                    ? `${option.selectedColor} ring-2 ring-blue-500 ring-offset-2`
                                    : `${option.color} hover:shadow-sm`
                                }`}
                              >
                                <div className="flex items-start gap-3">
                                  <option.icon className="h-5 w-5 mt-0.5 text-slate-600 dark:text-zinc-400" />
                                  <div className="text-left">
                                    <div className="text-sm font-medium">{option.label}</div>
                                    <div className="text-xs opacity-75 mt-0.5">{option.caption}</div>
                                  </div>
                                </div>
                                <div id={`stt-${option.value}-description`} className="sr-only">
                                  {option.description}. {option.caption}
                                </div>
                              </button>
                            ))}
                          </div>
                          <div id="stt-description" className="sr-only">
                            Choose a speech-to-text provider for your agent. Use arrow keys to navigate between options, space or enter to select.
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* Meeting Behavior Section */}
                    <div className="space-y-8">
                      <div className="space-y-2">
                        <div className="flex items-center gap-3">
                          <div className="p-2 bg-slate-100 dark:bg-zinc-800 rounded-lg">
                            <Target className="h-5 w-5 text-slate-600 dark:text-zinc-400" />
                          </div>
                          <div>
                            <h2 id="meeting-behavior-heading" className="text-2xl font-semibold">Meeting Behavior</h2>
                            <p className="text-sm text-slate-600 dark:text-zinc-400 mt-1">
                              Configure how your agent behaves in different meeting scenarios
                            </p>
                          </div>
                        </div>
                      </div>

                      {/* Response Behavior */}
                      <div className="space-y-4">
                        <div className="space-y-2">
                          <Label className="text-sm font-medium">Response Behavior</Label>
                          <p className="text-xs text-slate-500 dark:text-zinc-500">Control when and how your agent responds</p>
                        </div>
                        <div className="space-y-3">
                          <div className="flex items-start gap-3 p-4 bg-slate-50 dark:bg-zinc-900/50 rounded-lg border border-slate-200 dark:border-zinc-700">
                            <Checkbox
                              id="name_trigger"
                              {...register('name_trigger')}
                              className="mt-0.5"
                              aria-describedby="name-trigger-description"
                            />
                            <div className="space-y-1">
                              <Label htmlFor="name_trigger" className="text-sm font-medium cursor-pointer">
                                Name-based responses only
                              </Label>
                              <p id="name-trigger-description" className="text-xs text-slate-600 dark:text-zinc-400">
                                Agent will only respond when directly addressed by name, making conversations more focused and relevant
                              </p>
                            </div>
                          </div>
                          
                          <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-700">
                            <div className="flex items-start gap-3">
                              <div className="w-5 h-5 bg-blue-500 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5">
                                <span className="text-white text-xs">✓</span>
                              </div>
                              <div className="space-y-1">
                                <div className="text-sm font-medium text-blue-900 dark:text-blue-100">
                                  Conversational Context Enabled
                                </div>
                                <p className="text-xs text-blue-700 dark:text-blue-300">
                                  All agents maintain conversational context automatically to provide relevant, contextual responses
                                </p>
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>

                      {/* Custom Prompt Section */}
                      <PromptConfiguration
                        conversationMode={watchedConversationMode}
                        register={register}
                        errors={errors}
                        watch={watch}
                      />

                    </div>

                    {/* Advanced Parameters Section */}
                    <div className="space-y-8">
                      <div className="space-y-2">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <div className="p-2 bg-slate-100 dark:bg-zinc-800 rounded-lg">
                              <Settings className="h-5 w-5 text-slate-600 dark:text-zinc-400" />
                            </div>
                            <div>
                              <h2 id="advanced-parameters-heading" className="text-2xl font-semibold">Advanced Parameters</h2>
                              <p className="text-sm text-slate-600 dark:text-zinc-400 mt-1">
                                Fine-tune your agent&apos;s behavior with advanced settings
                              </p>
                            </div>
                          </div>
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            onClick={() => setShowAdvanced(!showAdvanced)}
                            className="gap-2"
                          >
                            {showAdvanced ? (
                              <>
                                <ChevronDown className="h-4 w-4" />
                                Hide Advanced
                              </>
                            ) : (
                              <>
                                <ChevronRight className="h-4 w-4" />
                                Show Advanced
                              </>
                            )}
                          </Button>
                        </div>
                      </div>

                      {/* Advanced Parameters Fields */}
                      {showAdvanced && (
                        <div className="space-y-6">
                          <div className="grid gap-6 grid-cols-1 sm:grid-cols-2">
                            {/* Utterance Tail Seconds */}
                            <div className="space-y-3">
                              <div className="space-y-2">
                                <Label htmlFor="utterance_tail_seconds" className="text-base font-semibold flex items-center gap-2 text-slate-900 dark:text-slate-100">
                                  Utterance Tail Seconds
                                </Label>
                                <p className="text-sm text-slate-600 dark:text-zinc-400">Duration to wait after speech ends before processing</p>
                              </div>
                              <Input
                                id="utterance_tail_seconds"
                                type="number"
                                step="0.1"
                                min="0.1"
                                max="5.0"
                                {...register('utterance_tail_seconds', { valueAsNumber: true })}
                                placeholder="e.g., 1.0"
                                className="h-12 text-base border-2 border-blue-200 dark:border-blue-700 focus:border-blue-500 dark:focus:border-blue-500 bg-white dark:bg-slate-900/50 transition-colors rounded-lg"
                                aria-describedby="utterance-tail-seconds-description"
                                aria-invalid={!!errors.utterance_tail_seconds}
                              />
                              <div id="utterance-tail-seconds-description" className="sr-only">Enter the duration in seconds</div>
                              {errors.utterance_tail_seconds && (
                                <p className="text-sm text-red-600" role="alert" aria-live="polite">
                                  {errors.utterance_tail_seconds.message}
                                </p>
                              )}
                            </div>

                            {/* No Speech Event Delay */}
                            <div className="space-y-3">
                              <div className="space-y-2">
                                <Label htmlFor="no_speech_event_delay" className="text-base font-semibold flex items-center gap-2 text-slate-900 dark:text-slate-100">
                                  No Speech Event Delay
                                </Label>
                                <p className="text-sm text-slate-600 dark:text-zinc-400">Delay before considering no speech event</p>
                              </div>
                              <Input
                                id="no_speech_event_delay"
                                type="number"
                                step="0.1"
                                min="0.1"
                                max="2.0"
                                {...register('no_speech_event_delay', { valueAsNumber: true })}
                                placeholder="e.g., 0.5"
                                className="h-12 text-base border-2 border-blue-200 dark:border-blue-700 focus:border-blue-500 dark:focus:border-blue-500 bg-white dark:bg-slate-900/50 transition-colors rounded-lg"
                                aria-describedby="no-speech-event-delay-description"
                                aria-invalid={!!errors.no_speech_event_delay}
                              />
                              <div id="no-speech-event-delay-description" className="sr-only">Enter the delay in seconds</div>
                              {errors.no_speech_event_delay && (
                                <p className="text-sm text-red-600" role="alert" aria-live="polite">
                                  {errors.no_speech_event_delay.message}
                                </p>
                              )}
                            </div>

                            {/* Max STT Tasks */}
                            <div className="space-y-3">
                              <div className="space-y-2">
                                <Label htmlFor="max_stt_tasks" className="text-base font-semibold flex items-center gap-2 text-slate-900 dark:text-slate-100">
                                  Max STT Tasks
                                </Label>
                                <p className="text-sm text-slate-600 dark:text-zinc-400">Maximum number of concurrent STT tasks</p>
                              </div>
                              <Input
                                id="max_stt_tasks"
                                type="number"
                                min="1"
                                max="20"
                                  {...register('max_stt_tasks', { valueAsNumber: true })}
                                placeholder="e.g., 5"
                                className="h-12 text-base border-2 border-blue-200 dark:border-blue-700 focus:border-blue-500 dark:focus:border-blue-500 bg-white dark:bg-slate-900/50 transition-colors rounded-lg"
                                aria-describedby="max-stt-tasks-description"
                                aria-invalid={!!errors.max_stt_tasks}
                              />
                              <div id="max-stt-tasks-description" className="sr-only">Enter the maximum number of STT tasks</div>
                              {errors.max_stt_tasks && (
                                <p className="text-sm text-red-600" role="alert" aria-live="polite">
                                  {errors.max_stt_tasks.message}
                                </p>
                              )}
                            </div>

                            {/* Window Queue Size */}
                            <div className="space-y-3">
                              <div className="space-y-2">
                                <Label htmlFor="window_queue_size" className="text-base font-semibold flex items-center gap-2 text-slate-900 dark:text-slate-100">
                                  Window Queue Size
                                </Label>
                                <p className="text-sm text-slate-600 dark:text-zinc-400">Size of the window queue for processing</p>
                              </div>
                              <Input
                                id="window_queue_size"
                                type="number"
                                min="10"
                                max="1000"
                                {...register('window_queue_size', { valueAsNumber: true })}
                                placeholder="e.g., 100"
                                className="h-12 text-base border-2 border-blue-200 dark:border-blue-700 focus:border-blue-500 dark:focus:border-blue-500 bg-white dark:bg-slate-900/50 transition-colors rounded-lg"
                                aria-describedby="window-queue-size-description"
                                aria-invalid={!!errors.window_queue_size}
                              />
                              <div id="window-queue-size-description" className="sr-only">Enter the size of the window queue</div>
                              {errors.window_queue_size && (
                                <p className="text-sm text-red-600" role="alert" aria-live="polite">
                                  {errors.window_queue_size.message}
                                </p>
                              )}
                            </div>
                          </div>

                          {/* STT Args */}
                          <div className="space-y-3">
                            <div className="space-y-2">
                              <Label htmlFor="stt_args" className="text-base font-semibold flex items-center gap-2 text-slate-900 dark:text-slate-100">
                                STT Args
                              </Label>
                              <p className="text-sm text-slate-600 dark:text-zinc-400">Additional arguments for STT processing</p>
                            </div>
                            <Textarea
                              id="stt_args"
                              {...register('stt_args')}
                              placeholder="e.g., { 'arg1': 'value1', 'arg2': 'value2' }"
                              rows={3}
                              className="font-mono text-sm"
                              aria-describedby="stt-args-description"
                            />
                            <div id="stt-args-description" className="sr-only">Enter additional arguments for STT processing</div>
                          </div>

                          {/* TTS Args */}
                          <div className="space-y-3">
                            <div className="space-y-2">
                              <Label htmlFor="tts_args" className="text-base font-semibold flex items-center gap-2 text-slate-900 dark:text-slate-100">
                                TTS Args
                              </Label>
                              <p className="text-sm text-slate-600 dark:text-zinc-400">Additional arguments for TTS processing</p>
                            </div>
                            <Textarea
                              id="tts_args"
                              {...register('tts_args')}
                              placeholder="e.g., { 'arg1': 'value1', 'arg2': 'value2' }"
                              rows={3}
                              className="font-mono text-sm"
                              aria-describedby="tts-args-description"
                            />
                            <div id="tts-args-description" className="sr-only">Enter additional arguments for TTS processing</div>
                          </div>

                          {/* VAD Args */}
                          <div className="space-y-3">
                            <div className="space-y-2">
                              <Label htmlFor="vad_args" className="text-base font-semibold flex items-center gap-2 text-slate-900 dark:text-slate-100">
                                VAD Args
                              </Label>
                              <p className="text-sm text-slate-600 dark:text-zinc-400">Additional arguments for VAD processing</p>
                            </div>
                            <Textarea
                              id="vad_args"
                              {...register('vad_args')}
                              placeholder="e.g., { 'arg1': 'value1', 'arg2': 'value2' }"
                              rows={3}
                              className="font-mono text-sm"
                              aria-describedby="vad-args-description"
                            />
                            <div id="vad-args-description" className="sr-only">Enter additional arguments for VAD processing</div>
                          </div>
                        </div>
                      )}
                    </div>

                {/* Create Button */}
                <div className="flex justify-end pt-8 border-t border-slate-200 dark:border-zinc-700">
                  <Button
                    type="submit"
                    disabled={isSubmitting}
                    size="lg"
                    className="px-8 py-3 bg-slate-900 hover:bg-slate-800 text-white dark:bg-slate-100 dark:hover:bg-slate-200 dark:text-slate-900"
                  >
                    {isSubmitting ? (
                      <>
                        <Loader2 className="h-5 w-5 mr-2 animate-spin" />
                        Creating Agent...
                      </>
                    ) : (
                      <>
                        <Save className="h-5 w-5 mr-2" />
                        Create Agent
                      </>
                    )}
                  </Button>
                </div>
              </form>
            </div>

            {/* Sticky Preview Pane */}
            <div className={`space-y-6 transition-all duration-300 ease-in-out xl:sticky xl:top-8 ${
              showPreview ? 'opacity-100' : 'opacity-0 xl:opacity-100'
            }`}
            style={{ animationDelay: `${previewKey * 50}ms` }}
            >
              <Card className="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/50 shadow-lg">
                <CardHeader className="pb-4">
                  <CardTitle className="flex items-center gap-2 text-lg">
                    <div className="p-1.5 bg-slate-100 dark:bg-zinc-800 rounded-md">
                      <Eye className="h-4 w-4 text-slate-600 dark:text-zinc-400" />
                    </div>
                    Live Preview
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  {/* Agent Avatar and Info */}
                  <div className="text-center space-y-4">
                    <div className="relative">
                      <div className="w-24 h-24 mx-auto rounded-full flex items-center justify-center transition-all duration-300 bg-gradient-to-br from-blue-100 to-purple-200 dark:from-blue-900/30 dark:to-purple-800/30">
                        <div className="text-3xl">🤖</div>
                      </div>

                      {/* Provider Badge */}
                      {selectedProvider && (
                        <div className="absolute -top-2 -right-2 w-8 h-8 bg-white dark:bg-zinc-800 rounded-full flex items-center justify-center shadow-lg border-2 border-white dark:border-zinc-700 overflow-hidden">
                          <Image
                            src={getProviderLogo(selectedProvider, isDarkMode, true)}
                            alt=""
                            width={24}
                            height={24}
                            className="object-contain"
                          />
                        </div>
                      )}

                      {/* Status Indicator */}
                      <div className="absolute -bottom-1 left-1/2 transform -translate-x-1/2">
                        <div className="w-4 h-4 bg-green-400 rounded-full border-2 border-white dark:border-zinc-800 shadow-sm"></div>
                      </div>
                    </div>

                    <div className="space-y-2">
                      <h3 className="text-xl font-bold">
                        {watch('name') || 'Your Agent'}
                      </h3>
                      <div className="text-sm text-muted-foreground">
                        {watch('conversation_mode') === 'analyst'
                          ? 'Silent Analysis & Comprehensive Note-Taking'
                          : 'AI Meeting Assistant with Conversational Context'
                        }
                      </div>

                      {/* Capability Badges */}
                      <div className="flex flex-wrap justify-center gap-1 mt-3">
                        {selectedSTT && (
                          <Badge variant="outline" className="text-xs px-2 py-1">
                            <Ear className="h-3 w-3 inline mr-1" />
                            {selectedSTT.label}
                          </Badge>
                        )}
                        {watch('conversation_mode') === 'conversational' && selectedTTS && (
                          <Badge variant="outline" className="text-xs px-2 py-1">
                            <Volume2 className="h-3 w-3 inline mr-1" />
                            {selectedTTS.label}
                          </Badge>
                        )}
                        {selectedLanguage && (
                          <Badge variant="outline" className="text-xs px-2 py-1">
                            {selectedLanguage.flag} {selectedLanguage.label}
                          </Badge>
                        )}
                        <Badge variant="outline" className="text-xs px-2 py-1 bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-700">
                          {watch('conversation_mode') === 'analyst' ? (
                            <>
                              <Eye className="h-3 w-3 inline mr-1" />
                              Analyst Mode
                            </>
                          ) : (
                            <>
                              <Brain className="h-3 w-3 inline mr-1" />
                              Context Aware
                            </>
                          )}
                        </Badge>
                      </div>
                    </div>
                  </div>

                  {/* Agent Workflow */}
                  <div className="space-y-4">
                    <h4 className="text-sm font-semibold text-center">Agent Workflow</h4>
                    <div className="relative">
                      <div className="absolute left-6 top-8 bottom-8 w-0.5 bg-gradient-to-b from-slate-200 via-slate-300 to-slate-200 dark:from-zinc-600 dark:via-zinc-500 dark:to-zinc-600"></div>
                      <div className="space-y-1">
                        {(() => {
                          const selectedMode = CONVERSATION_MODE_OPTIONS.find(mode => mode.value === watch('conversation_mode'));
                          return selectedMode ? selectedMode.workflow.map((step, idx) => (
                            <div key={idx} className="flex items-center gap-3">
                              <div className={`w-3 h-3 ${step.color} rounded-full flex-shrink-0`}></div>
                              <span className="text-sm">{step.step}</span>
                            </div>
                          )) : null;
                        })()}
                      </div>
                    </div>

                    {/* Configuration Insights */}
                    <div className="mt-4 p-3 bg-gradient-to-r from-slate-50 to-slate-100 dark:from-zinc-900/50 dark:to-zinc-800/50 rounded-lg border border-slate-200 dark:border-zinc-700">
                      <div className="text-center">
                        <div className="text-sm font-medium mb-1">
                          {watch('conversation_mode') === 'analyst'
                            ? '📊 Comprehensive Meeting Analysis'
                            : '🎯 Intelligent Meeting Assistant'
                          }
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {watch('conversation_mode') === 'analyst'
                            ? 'Advanced AI analysis for detailed meeting insights and structured reporting'
                            : 'Configured with conversational context for natural, contextual responses'
                          }
                        </div>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}

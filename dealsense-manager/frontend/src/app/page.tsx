/**
 * Standalone landing page for Joinly AI - no sidebar, single page design
 */

'use client';

import { useRouter } from 'next/navigation';
import {
  ArrowRight,
  Bot,
  Brain,
  CheckCircle,
  Globe,
  MessageSquare,
  Shield,
  Star,
  Users,
  Volume2,
  Zap,
  Eye,
  Play,
  Clock,
  Headphones
} from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';

export default function LandingPage() {
  const router = useRouter();

  return (
    <div className="min-h-screen bg-gray-900">
      {/* Header */}
      <header className="relative bg-gray-900 border-b border-gray-800">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-6">
            <div className="flex items-center">
              <div className="flex items-center space-x-3">
                <div className="w-10 h-10 bg-gradient-to-r from-blue-600 to-purple-600 rounded-xl flex items-center justify-center">
                  <Bot className="w-6 h-6 text-white" />
                </div>
                <h1 className="text-2xl font-bold text-white">
                  DealSense AI
                </h1>
              </div>
            </div>
            <nav className="hidden md:flex space-x-8">
              <a href="#features" className="text-gray-300 hover:text-white transition-colors">
                Features
              </a>
              <a href="#how-it-works" className="text-gray-300 hover:text-white transition-colors">
                How it Works
              </a>
              <a href="#benefits" className="text-gray-300 hover:text-white transition-colors">
                Benefits
              </a>
            </nav>
            <div className="flex items-center space-x-4">
              <Button
                onClick={() => router.push('/agents')}
                variant="outline"
                className="hidden sm:inline-flex"
              >
                View Agents
              </Button>
              <Button
                onClick={() => router.push('/agents/create')}
                className="bg-blue-600 hover:bg-blue-700"
              >
                Get Started
              </Button>
            </div>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative py-20 lg:py-32">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="lg:grid lg:grid-cols-12 lg:gap-12 items-center">
            <div className="lg:col-span-6">
              <h1 className="text-4xl md:text-5xl lg:text-6xl font-bold text-white leading-tight">
                AI That Joins Your
                <span className="text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-purple-400">
                  {" "}Meetings
                </span>
              </h1>
              <p className="mt-6 text-xl text-gray-300 leading-relaxed">
                Deploy intelligent AI agents that participate naturally in your meetings. 
                Get real-time transcription, intelligent responses, and comprehensive analysis 
                without the complexity.
              </p>
              <div className="mt-8 flex flex-col sm:flex-row gap-4">
                <Button
                  onClick={() => router.push('/agents/create')}
                  size="lg"
                  className="bg-blue-600 hover:bg-blue-700 text-white px-8 py-4 text-lg"
                >
                  Get Started Free
                  <ArrowRight className="ml-2 w-5 h-5" />
                </Button>
                <Button
                  onClick={() => router.push('/meetings')}
                  variant="outline"
                  size="lg"
                  className="px-8 py-4 text-lg"
                >
                  <Play className="mr-2 w-5 h-5" />
                  Watch Demo
                </Button>
              </div>
              <div className="mt-8 flex items-center space-x-6 text-sm text-gray-400">
                <div className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-400 mr-2" />
                  No credit card required
                </div>
                <div className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-400 mr-2" />
                  Setup in minutes
                </div>
                <div className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-400 mr-2" />
                  Privacy first
                </div>
              </div>
            </div>
            <div className="mt-12 lg:mt-0 lg:col-span-6">
              <div className="relative">
                <div className="aspect-square bg-gradient-to-br from-gray-800 to-gray-700 rounded-3xl p-8 flex items-center justify-center">
                  <div className="grid grid-cols-2 gap-6 w-full max-w-sm">
                    <Card className="bg-gray-800 shadow-lg border border-gray-700">
                      <CardContent className="p-6">
                        <div className="w-12 h-12 bg-blue-900/50 rounded-lg flex items-center justify-center mb-4">
                          <Bot className="w-6 h-6 text-blue-400" />
                        </div>
                        <h3 className="font-semibold text-white mb-2">AI Agent</h3>
                        <p className="text-sm text-gray-300">Intelligent meeting participant</p>
                      </CardContent>
                    </Card>
                    <Card className="bg-gray-800 shadow-lg border border-gray-700 mt-8">
                      <CardContent className="p-6">
                        <div className="w-12 h-12 bg-green-900/50 rounded-lg flex items-center justify-center mb-4">
                          <MessageSquare className="w-6 h-6 text-green-400" />
                        </div>
                        <h3 className="font-semibold text-white mb-2">Smart Responses</h3>
                        <p className="text-sm text-gray-300">Context-aware conversations</p>
                      </CardContent>
                    </Card>
                    <Card className="bg-gray-800 shadow-lg border border-gray-700 -mt-4">
                      <CardContent className="p-6">
                        <div className="w-12 h-12 bg-purple-900/50 rounded-lg flex items-center justify-center mb-4">
                          <Eye className="w-6 h-6 text-purple-400" />
                        </div>
                        <h3 className="font-semibold text-white mb-2">Meeting Analysis</h3>
                        <p className="text-sm text-gray-300">Comprehensive insights</p>
                      </CardContent>
                    </Card>
                    <Card className="bg-gray-800 shadow-lg border border-gray-700 mt-4">
                      <CardContent className="p-6">
                        <div className="w-12 h-12 bg-orange-900/50 rounded-lg flex items-center justify-center mb-4">
                          <Shield className="w-6 h-6 text-orange-400" />
                        </div>
                        <h3 className="font-semibold text-white mb-2">Privacy Secure</h3>
                        <p className="text-sm text-gray-300">Your data stays safe</p>
                      </CardContent>
                    </Card>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section id="features" className="py-20 bg-gray-800">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">
              Everything you need for intelligent meetings
            </h2>
            <p className="text-xl text-gray-300 max-w-2xl mx-auto">
              Deploy AI agents that understand context, participate naturally, and provide valuable insights.
            </p>
          </div>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-8">
            <Card className="bg-gray-700 border border-gray-600 shadow-sm hover:shadow-md transition-shadow">
              <CardContent className="p-8">
                <div className="w-12 h-12 bg-blue-900/50 rounded-lg flex items-center justify-center mb-6">
                  <MessageSquare className="w-6 h-6 text-blue-400" />
                </div>
                <h3 className="text-xl font-semibold text-white mb-3">Conversational Mode</h3>
                <p className="text-gray-300">
                  AI agents that actively participate in meetings with natural, contextual responses that add real value to discussions.
                </p>
              </CardContent>
            </Card>
            <Card className="bg-gray-700 border border-gray-600 shadow-sm hover:shadow-md transition-shadow">
              <CardContent className="p-8">
                <div className="w-12 h-12 bg-green-900/50 rounded-lg flex items-center justify-center mb-6">
                  <Eye className="w-6 h-6 text-green-400" />
                </div>
                <h3 className="text-xl font-semibold text-white mb-3">Analyst Mode</h3>
                <p className="text-gray-300">
                  Silent observers that provide comprehensive analysis, meeting summaries, and actionable insights without interrupting.
                </p>
              </CardContent>
            </Card>
            <Card className="bg-gray-700 border border-gray-600 shadow-sm hover:shadow-md transition-shadow">
              <CardContent className="p-8">
                <div className="w-12 h-12 bg-purple-900/50 rounded-lg flex items-center justify-center mb-6">
                  <Volume2 className="w-6 h-6 text-purple-400" />
                </div>
                <h3 className="text-xl font-semibold text-white mb-3">Natural Speech</h3>
                <p className="text-gray-300">
                  Advanced text-to-speech with multiple providers for realistic, human-like voices that feel natural in conversations.
                </p>
              </CardContent>
            </Card>
            <Card className="bg-gray-700 border border-gray-600 shadow-sm hover:shadow-md transition-shadow">
              <CardContent className="p-8">
                <div className="w-12 h-12 bg-orange-900/50 rounded-lg flex items-center justify-center mb-6">
                  <Headphones className="w-6 h-6 text-orange-400" />
                </div>
                <h3 className="text-xl font-semibold text-white mb-3">Smart Listening</h3>
                <p className="text-gray-300">
                  State-of-the-art speech recognition with Whisper and Deepgram that understands context and nuance.
                </p>
              </CardContent>
            </Card>
            <Card className="bg-gray-700 border border-gray-600 shadow-sm hover:shadow-md transition-shadow">
              <CardContent className="p-8">
                <div className="w-12 h-12 bg-indigo-900/50 rounded-lg flex items-center justify-center mb-6">
                  <Globe className="w-6 h-6 text-indigo-400" />
                </div>
                <h3 className="text-xl font-semibold text-white mb-3">Multi-Platform</h3>
                <p className="text-gray-300">
                  Works seamlessly with Google Meet, Zoom, Microsoft Teams, and other major video conferencing platforms.
                </p>
              </CardContent>
            </Card>
            <Card className="bg-gray-700 border border-gray-600 shadow-sm hover:shadow-md transition-shadow">
              <CardContent className="p-8">
                <div className="w-12 h-12 bg-red-900/50 rounded-lg flex items-center justify-center mb-6">
                  <Shield className="w-6 h-6 text-red-400" />
                </div>
                <h3 className="text-xl font-semibold text-white mb-3">Privacy First</h3>
                <p className="text-gray-300">
                  Your meeting data is processed securely with enterprise-grade privacy protections and local processing options.
                </p>
              </CardContent>
            </Card>
          </div>
        </div>
      </section>

      {/* How it Works Section */}
      <section id="how-it-works" className="py-20 bg-gray-900">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">
              Simple setup, powerful results
            </h2>
            <p className="text-xl text-gray-300 max-w-2xl mx-auto">
              Get your AI meeting assistant up and running in just a few clicks.
            </p>
          </div>
          <div className="grid md:grid-cols-3 gap-8">
            <div className="text-center">
              <div className="w-16 h-16 bg-blue-900/50 rounded-full flex items-center justify-center mx-auto mb-6">
                <span className="text-2xl font-bold text-blue-400">1</span>
              </div>
              <h3 className="text-xl font-semibold text-white mb-3">Create Your Agent</h3>
              <p className="text-gray-300">
                Choose your AI provider, configure voice settings, and customize your agent&apos;s behavior in minutes.
              </p>
            </div>
            <div className="text-center">
              <div className="w-16 h-16 bg-green-900/50 rounded-full flex items-center justify-center mx-auto mb-6">
                <span className="text-2xl font-bold text-green-400">2</span>
              </div>
              <h3 className="text-xl font-semibold text-white mb-3">Add Meeting URL</h3>
              <p className="text-gray-300">
                Simply paste your Google Meet, Zoom, or Teams link and your agent will automatically join when ready.
              </p>
            </div>
            <div className="text-center">
              <div className="w-16 h-16 bg-purple-900/50 rounded-full flex items-center justify-center mx-auto mb-6">
                <span className="text-2xl font-bold text-purple-400">3</span>
              </div>
              <h3 className="text-xl font-semibold text-white mb-3">Start Meeting</h3>
              <p className="text-gray-300">
                Your AI agent joins seamlessly and begins providing value immediately with intelligent participation.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Benefits Section */}
      <section id="benefits" className="py-20 bg-gradient-to-br from-gray-800 to-gray-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="lg:grid lg:grid-cols-2 lg:gap-16 items-center">
            <div>
              <h2 className="text-3xl md:text-4xl font-bold text-white mb-6">
                Why teams choose DealSense
              </h2>
              <div className="space-y-6">
                <div className="flex items-start">
                  <div className="w-6 h-6 bg-blue-900/50 rounded-full flex items-center justify-center mr-4 mt-1">
                    <Clock className="w-4 h-4 text-blue-400" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-white mb-2">Save Time & Focus</h3>
                    <p className="text-gray-300">
                      Let AI handle note-taking and follow-ups while you focus on the conversation that matters.
                    </p>
                  </div>
                </div>
                <div className="flex items-start">
                  <div className="w-6 h-6 bg-green-900/50 rounded-full flex items-center justify-center mr-4 mt-1">
                    <Brain className="w-4 h-4 text-green-400" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-white mb-2">Intelligent Insights</h3>
                    <p className="text-gray-300">
                      Get actionable meeting summaries, key decisions, and follow-up tasks automatically generated.
                    </p>
                  </div>
                </div>
                <div className="flex items-start">
                  <div className="w-6 h-6 bg-purple-900/50 rounded-full flex items-center justify-center mr-4 mt-1">
                    <Users className="w-4 h-4 text-purple-400" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-white mb-2">Better Collaboration</h3>
                    <p className="text-gray-300">
                      AI agents help bridge communication gaps and ensure everyone stays aligned and informed.
                    </p>
                  </div>
                </div>
                <div className="flex items-start">
                  <div className="w-6 h-6 bg-orange-900/50 rounded-full flex items-center justify-center mr-4 mt-1">
                    <Zap className="w-4 h-4 text-orange-400" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-white mb-2">Instant Setup</h3>
                    <p className="text-gray-300">
                      Deploy in minutes, not days. No complex integrations or lengthy onboarding required.
                    </p>
                  </div>
                </div>
              </div>
            </div>
            <div className="mt-12 lg:mt-0">
              <Card className="bg-gray-700 shadow-xl border border-gray-600 p-8">
                <CardContent className="p-0">
                  <div className="text-center mb-6">
                    <div className="w-16 h-16 bg-gradient-to-r from-blue-600 to-purple-600 rounded-full flex items-center justify-center mx-auto mb-4">
                      <Star className="w-8 h-8 text-white" />
                    </div>
                    <h3 className="text-2xl font-bold text-white mb-2">Ready to get started?</h3>
                    <p className="text-gray-300">
                      Join thousands of teams already using AI to supercharge their meetings.
                    </p>
                  </div>
                  <Button
                    onClick={() => router.push('/agents/create')}
                    size="lg"
                    className="w-full bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white py-4"
                  >
                    Get Started Free
                    <ArrowRight className="ml-2 w-5 h-5" />
                  </Button>
                </CardContent>
              </Card>
            </div>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-black text-white py-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid md:grid-cols-4 gap-8">
            <div className="md:col-span-2">
              <div className="flex items-center space-x-3 mb-4">
                <div className="w-10 h-10 bg-gradient-to-r from-blue-600 to-purple-600 rounded-xl flex items-center justify-center">
                  <Bot className="w-6 h-6 text-white" />
                </div>
                <h3 className="text-xl font-bold">DealSense AI</h3>
              </div>
              <p className="text-gray-400 mb-4 max-w-md">
                Intelligent AI agents that transform how teams collaborate in meetings. 
                Get more done with less effort.
              </p>
            </div>
            <div>
              <h4 className="font-semibold mb-4">Product</h4>
              <ul className="space-y-2 text-gray-400">
                <li>
                  <button 
                    onClick={() => router.push('/agents')}
                    className="hover:text-white transition-colors"
                  >
                    Agents
                  </button>
                </li>
                <li>
                  <button 
                    onClick={() => router.push('/meetings')}
                    className="hover:text-white transition-colors"
                  >
                    Meetings
                  </button>
                </li>
                <li>
                  <button 
                    onClick={() => router.push('/agents/create')}
                    className="hover:text-white transition-colors"
                  >
                    Get Started
                  </button>
                </li>
              </ul>
            </div>
            <div>
              <h4 className="font-semibold mb-4">Company</h4>
              <ul className="space-y-2 text-gray-400">
                <li><a href="#" className="hover:text-white transition-colors">About</a></li>
                <li><a href="#" className="hover:text-white transition-colors">Privacy</a></li>
                <li><a href="#" className="hover:text-white transition-colors">Terms</a></li>
                <li><a href="#" className="hover:text-white transition-colors">Support</a></li>
              </ul>
            </div>
          </div>
          <div className="border-t border-gray-800 mt-8 pt-8 text-center text-gray-400">
            <p>&copy; 2025 DealSense AI. All rights reserved.</p>
          </div>
        </div>
      </footer>
    </div>
  );
}
/**
 * Startup Analysis component displaying multi-faceted investment analysis
 */

'use client';

import { useState, useEffect, useCallback } from 'react';
import { TrendingUp, AlertTriangle, Lightbulb, CheckCircle2, XCircle, RefreshCw, Loader2, Target, DollarSign, Users, Zap } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { analysisApi, StartupAnalysisResult, Document } from '@/lib/api';

interface StartupAnalysisProps {
  agentId: string;
  documents: Document[];
  onAnalyze?: () => void;
}

export function StartupAnalysis({ agentId, documents, onAnalyze }: StartupAnalysisProps) {
  const [analysis, setAnalysis] = useState<StartupAnalysisResult | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isAnalyzing, setIsAnalyzing] = useState(false);

  const loadAnalysis = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await analysisApi.getLatest(agentId);
      setAnalysis(response.data);
    } catch (error) {
      console.error('Failed to load analysis:', error);
      setAnalysis(null);
    } finally {
      setIsLoading(false);
    }
  }, [agentId]);

  useEffect(() => {
    loadAnalysis();
  }, [loadAnalysis]);

  const handleAnalyze = async () => {
    if (documents.length === 0) {
      alert('Please upload at least one document to analyze');
      return;
    }

    const processedDocs = documents.filter(d => d.status === 'processed');
    if (processedDocs.length === 0) {
      alert('Please wait for documents to finish processing');
      return;
    }

    setIsAnalyzing(true);
    try {
      const response = await analysisApi.analyze(agentId, {
        document_ids: processedDocs.map(d => d.id),
        analysis_types: [
          'pitch_analysis',
          'founder_reliability',
          'market_opportunity',
          'financial_viability',
          'competitive_landscape',
        ],
      });
      setAnalysis(response.data);
      onAnalyze?.();
    } catch (error) {
      console.error('Analysis failed:', error);
      alert('Failed to analyze startup. Please try again.');
    } finally {
      setIsAnalyzing(false);
    }
  };

  const getScoreColor = (score: number) => {
    if (score >= 80) return 'text-green-600 dark:text-green-400';
    if (score >= 65) return 'text-blue-600 dark:text-blue-400';
    if (score >= 50) return 'text-yellow-600 dark:text-yellow-400';
    return 'text-red-600 dark:text-red-400';
  };

  const getScoreBgColor = (score: number) => {
    if (score >= 80) return 'bg-green-100 dark:bg-green-900/30';
    if (score >= 65) return 'bg-blue-100 dark:bg-blue-900/30';
    if (score >= 50) return 'bg-yellow-100 dark:bg-yellow-900/30';
    return 'bg-red-100 dark:bg-red-900/30';
  };

  const getRecommendation = (score: number) => {
    if (score >= 80) return { text: 'STRONG INVEST', color: 'bg-green-600', icon: <CheckCircle2 className="h-5 w-5" /> };
    if (score >= 65) return { text: 'CONSIDER', color: 'bg-blue-600', icon: <Target className="h-5 w-5" /> };
    if (score >= 50) return { text: 'CAUTIOUS', color: 'bg-yellow-600', icon: <AlertTriangle className="h-5 w-5" /> };
    return { text: 'PASS', color: 'bg-red-600', icon: <XCircle className="h-5 w-5" /> };
  };

  const getSectionIcon = (type: string) => {
    switch (type) {
      case 'pitch_analysis':
        return <Target className="h-5 w-5" />;
      case 'founder_reliability':
        return <Users className="h-5 w-5" />;
      case 'market_opportunity':
        return <TrendingUp className="h-5 w-5" />;
      case 'financial_viability':
        return <DollarSign className="h-5 w-5" />;
      case 'competitive_landscape':
        return <Zap className="h-5 w-5" />;
      default:
        return <Target className="h-5 w-5" />;
    }
  };

  const formatSectionTitle = (type: string) => {
    return type.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ');
  };

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
        </CardContent>
      </Card>
    );
  }

  if (!analysis) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrendingUp className="h-5 w-5" />
            Startup Analysis
          </CardTitle>
          <CardDescription>
            AI-powered multi-faceted analysis of pitch deck and meeting data
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8">
            <TrendingUp className="h-16 w-16 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
              No Analysis Available
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-6">
              Upload documents and run analysis to get comprehensive startup insights
            </p>
            <Button
              onClick={handleAnalyze}
              disabled={isAnalyzing || documents.length === 0}
              size="lg"
            >
              {isAnalyzing ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Analyzing...
                </>
              ) : (
                <>
                  <TrendingUp className="h-4 w-4 mr-2" />
                  Run Analysis
                </>
              )}
            </Button>
            {documents.length === 0 && (
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-3">
                Upload at least one document to start
              </p>
            )}
          </div>
        </CardContent>
      </Card>
    );
  }

  const recommendation = getRecommendation(analysis.overall_score);

  return (
    <div className="space-y-6">
      {/* Overall Score Card */}
      <Card className="border-l-4 border-l-blue-500">
        <CardContent className="p-6">
          <div className="flex items-center justify-between">
            <div className="flex-1">
              <div className="flex items-center gap-3 mb-4">
                <div className={`${recommendation.color} text-white p-3 rounded-lg`}>
                  {recommendation.icon}
                </div>
                <div>
                  <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
                    Investment Recommendation: {recommendation.text}
                  </h2>
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    Analysis completed {new Date(analysis.generated_at).toLocaleString()}
                  </p>
                </div>
              </div>
              
              <div className="bg-gray-50 dark:bg-gray-900/50 rounded-lg p-4 mb-4">
                <p className="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap">
                  {analysis.summary}
                </p>
              </div>
            </div>

            <div className="ml-8 text-center">
              <div className={`${getScoreBgColor(analysis.overall_score)} rounded-full p-6 mb-3`}>
                <div className={`text-5xl font-bold ${getScoreColor(analysis.overall_score)}`}>
                  {analysis.overall_score.toFixed(1)}
                </div>
                <div className="text-sm font-medium text-gray-600 dark:text-gray-400">
                  Overall Score
                </div>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleAnalyze}
                disabled={isAnalyzing}
              >
                {isAnalyzing ? (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                ) : (
                  <RefreshCw className="h-4 w-4 mr-2" />
                )}
                Re-analyze
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Detailed Analysis Sections */}
      <Card>
        <CardHeader>
          <CardTitle>Detailed Analysis</CardTitle>
          <CardDescription>
            In-depth evaluation across multiple dimensions
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue={Object.keys(analysis.analysis_sections)[0]} className="w-full">
            <TabsList className="grid w-full grid-cols-5 mb-6">
              {Object.entries(analysis.analysis_sections).map(([type, section]) => (
                <TabsTrigger key={type} value={type} className="text-xs">
                  {formatSectionTitle(type)}
                </TabsTrigger>
              ))}
            </TabsList>

            {Object.entries(analysis.analysis_sections).map(([type, section]) => (
              <TabsContent key={type} value={type} className="space-y-6">
                {/* Section Score */}
                <div className="flex items-center gap-4 pb-4 border-b border-gray-200 dark:border-gray-800">
                  <div className="flex items-center gap-3">
                    <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                      {getSectionIcon(type)}
                    </div>
                    <div>
                      <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                        {formatSectionTitle(type)}
                      </h3>
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        {section.summary}
                      </p>
                    </div>
                  </div>
                  <div className="ml-auto">
                    <div className={`text-3xl font-bold ${getScoreColor(section.score)}`}>
                      {section.score.toFixed(1)}
                    </div>
                    <div className="text-xs text-gray-500 dark:text-gray-400 text-center">
                      Score
                    </div>
                  </div>
                </div>

                {/* Key Findings */}
                {section.key_findings.length > 0 && (
                  <div>
                    <div className="flex items-center gap-2 mb-3">
                      <CheckCircle2 className="h-5 w-5 text-green-600" />
                      <h4 className="font-semibold text-gray-900 dark:text-white">
                        Key Findings
                      </h4>
                    </div>
                    <ul className="space-y-2">
                      {section.key_findings.map((finding, index) => (
                        <li
                          key={index}
                          className="flex items-start gap-2 text-sm text-gray-700 dark:text-gray-300"
                        >
                          <span className="text-green-600 dark:text-green-400 mt-0.5">•</span>
                          <span>{finding}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {/* Red Flags */}
                {section.red_flags.length > 0 && (
                  <div>
                    <div className="flex items-center gap-2 mb-3">
                      <AlertTriangle className="h-5 w-5 text-red-600" />
                      <h4 className="font-semibold text-gray-900 dark:text-white">
                        Red Flags & Concerns
                      </h4>
                    </div>
                    <ul className="space-y-2">
                      {section.red_flags.map((flag, index) => (
                        <li
                          key={index}
                          className="flex items-start gap-2 text-sm text-gray-700 dark:text-gray-300"
                        >
                          <span className="text-red-600 dark:text-red-400 mt-0.5">⚠</span>
                          <span>{flag}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {/* Opportunities */}
                {section.opportunities.length > 0 && (
                  <div>
                    <div className="flex items-center gap-2 mb-3">
                      <Lightbulb className="h-5 w-5 text-yellow-600" />
                      <h4 className="font-semibold text-gray-900 dark:text-white">
                        Opportunities
                      </h4>
                    </div>
                    <ul className="space-y-2">
                      {section.opportunities.map((opportunity, index) => (
                        <li
                          key={index}
                          className="flex items-start gap-2 text-sm text-gray-700 dark:text-gray-300"
                        >
                          <span className="text-yellow-600 dark:text-yellow-400 mt-0.5">💡</span>
                          <span>{opportunity}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {/* Recommendations */}
                {section.recommendations.length > 0 && (
                  <div>
                    <div className="flex items-center gap-2 mb-3">
                      <Target className="h-5 w-5 text-blue-600" />
                      <h4 className="font-semibold text-gray-900 dark:text-white">
                        Recommendations
                      </h4>
                    </div>
                    <ul className="space-y-2">
                      {section.recommendations.map((recommendation, index) => (
                        <li
                          key={index}
                          className="flex items-start gap-2 text-sm text-gray-700 dark:text-gray-300"
                        >
                          <span className="text-blue-600 dark:text-blue-400 mt-0.5">→</span>
                          <span>{recommendation}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </TabsContent>
            ))}
          </Tabs>
        </CardContent>
      </Card>
    </div>
  );
}


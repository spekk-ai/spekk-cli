import fs from 'node:fs';
import path from 'node:path';
import { Skill } from './skill-interface.js';

export class BusinessModelValidator extends Skill {
  constructor() {
    super();
    this.assessmentAreas = [
      'industry-background',
      'product-validation', 
      'market-demand',
      'business-goals',
      'traction-momentum',
      'hypothesis-prioritization'
    ];

    this.questions = {
      'industry-background': [
        'What relevant experience does the founder have?',
        'How many years has the founder worked in this industry?',
        'What specific domain expertise does the team bring?',
        'Have any team members built similar products before?'
      ],
      'product-validation': [
        'What stage of development is the product in?',
        'Have you conducted any user testing?',
        'What feedback have you received from potential users?',
        'Do you have a working prototype or MVP?'
      ],
      'market-demand': [
        'What evidence exists for market demand?',
        'Who is your target customer?',
        'What problem are you solving for them?',
        'How big is the potential market?'
      ],
      'business-goals': [
        'What is your primary business goal?',
        'What is your monetization strategy?',
        'How do you plan to generate revenue?',
        'What are your key success metrics?'
      ],
      'traction-momentum': [
        'What are your current metrics?',
        'How much funding have you raised?',
        'Do you have paying customers?',
        'What growth have you seen recently?'
      ],
      'hypothesis-prioritization': [
        'What are your riskiest assumptions?',
        'What could invalidate your business model?',
        'Which hypotheses should you test first?',
        'How will you validate your key assumptions?'
      ]
    };

    this.scoreDescriptions = {
      2: 'clear/evidence',
      1: 'adequate',
      0: 'unclear/concerning'
    };

    this.recommendationTemplates = {
      'industry-background': {
        0: { priority: 'high', message: 'Build domain expertise through research, advisors, or partnerships' },
        1: { priority: 'medium', message: 'Strengthen industry knowledge and network connections' }
      },
      'product-validation': {
        0: { priority: 'high', message: 'Conduct user interviews and build a testable prototype' },
        1: { priority: 'medium', message: 'Expand user testing and gather more feedback data' }
      },
      'market-demand': {
        0: { priority: 'high', message: 'Research market size and validate customer pain points' },
        1: { priority: 'medium', message: 'Quantify market opportunity and refine customer segmentation' }
      },
      'business-goals': {
        0: { priority: 'high', message: 'Define clear business model and revenue strategy' },
        1: { priority: 'medium', message: 'Clarify success metrics and milestone planning' }
      },
      'traction-momentum': {
        0: { priority: 'high', message: 'Focus on customer acquisition and early traction metrics' },
        1: { priority: 'medium', message: 'Scale current traction and improve key metrics' }
      },
      'hypothesis-prioritization': {
        0: { priority: 'high', message: 'Identify and test your riskiest assumptions first' },
        1: { priority: 'medium', message: 'Prioritize hypotheses by impact and validation difficulty' }
      }
    };
  }

  // Skill interface implementation
  getId() {
    return 'business-model-validator';
  }

  getName() {
    return 'Business Model Validator';
  }

  getDescription() {
    return 'Validates startup/business models through structured questioning and scoring across key areas like product validation, market demand, and business strategy.';
  }

  shouldTrigger(userInput, context = {}) {
    const keywords = [
      'business model', 'startup', 'validate', 'business idea',
      'startup idea', 'business validation', 'market validation',
      'product validation', 'business strategy', 'business plan',
      'monetization', 'revenue model', 'traction', 'market demand'
    ];
    
    const lowerInput = userInput.toLowerCase();
    return keywords.some(keyword => lowerInput.includes(keyword));
  }

  getQuestions() {
    const questions = [];
    
    this.assessmentAreas.forEach(area => {
      const areaQuestions = this.getQuestionsForArea(area);
      areaQuestions.forEach((questionText, index) => {
        questions.push({
          id: `${area}_q${index + 1}`,
          text: questionText,
          type: 'text',
          area: area
        });
      });
    });
    
    return questions;
  }

  processResponses(responses) {
    // Create a session to use existing methods
    const session = this.createSession();
    
    // Group responses by area and record them
    this.assessmentAreas.forEach(area => {
      const areaResponses = [];
      const areaQuestions = this.getQuestionsForArea(area);
      
      areaQuestions.forEach((_, index) => {
        const responseKey = `${area}_q${index + 1}`;
        if (responses[responseKey]) {
          areaResponses.push(responses[responseKey]);
        }
      });
      
      if (areaResponses.length > 0) {
        this.recordResponse(session, area, areaResponses.join(' '));
        // For now, auto-score based on response completeness
        // In a real implementation, this would involve Claude analyzing the responses
        const score = this.autoScore(areaResponses);
        this.recordScore(session, area, score);
      }
    });
    
    // Generate the report
    const report = this.generateReport(session);
    const structuredReport = this.generateStructuredReport(session);
    
    return {
      summary: report.summary,
      recommendations: report.recommendations,
      data: {
        totalScore: report.totalScore,
        percentageScore: report.percentageScore,
        areaScores: report.areaScores,
        responses: report.responses
      },
      markdownReport: structuredReport
    };
  }

  // Helper method to auto-score based on response completeness (simplified)
  autoScore(responses) {
    const totalLength = responses.join(' ').length;
    const avgLength = totalLength / responses.length;
    
    if (avgLength > 100) return 2; // Detailed responses
    if (avgLength > 50) return 1;  // Adequate responses
    return 0; // Minimal responses
  }

  getAssessmentAreas() {
    return [...this.assessmentAreas];
  }

  getQuestionsForArea(area) {
    return this.questions[area] || [];
  }

  isValidScore(score) {
    return typeof score === 'number' && [0, 1, 2].includes(score);
  }

  calculateTotalScore(scores) {
    return Object.values(scores).reduce((sum, score) => sum + score, 0);
  }

  calculatePercentageScore(scores) {
    const total = this.calculateTotalScore(scores);
    const maxPossible = this.assessmentAreas.length * 2; // 6 areas * 2 max points each = 12
    return Math.round((total / maxPossible) * 100);
  }

  createSession() {
    return {
      responses: {},
      scores: {},
      currentArea: null,
      isComplete: false,
      startedAt: new Date().toISOString()
    };
  }

  recordResponse(session, area, response) {
    if (!this.assessmentAreas.includes(area)) {
      throw new Error(`Invalid assessment area: ${area}`);
    }
    session.responses[area] = response;
  }

  recordScore(session, area, score) {
    if (!this.assessmentAreas.includes(area)) {
      throw new Error(`Invalid assessment area: ${area}`);
    }
    if (!this.isValidScore(score)) {
      throw new Error(`Invalid score: ${score}. Must be 0, 1, or 2`);
    }
    session.scores[area] = score;
  }

  isSessionComplete(session) {
    return this.assessmentAreas.every(area => area in session.scores);
  }

  generateRecommendations(scores) {
    const recommendations = [];
    
    // Sort areas by score (lowest first) to prioritize recommendations
    const sortedAreas = this.assessmentAreas
      .map(area => ({ area, score: scores[area] || 0 }))
      .sort((a, b) => a.score - b.score);

    sortedAreas.forEach(({ area, score }) => {
      if (score < 2) { // Only recommend for areas that aren't at max score
        const template = this.recommendationTemplates[area][score];
        if (template) {
          recommendations.push({
            area,
            score,
            priority: template.priority,
            message: template.message
          });
        }
      }
    });

    return recommendations;
  }

  generateReport(session) {
    if (!this.isSessionComplete(session)) {
      throw new Error('Cannot generate report for incomplete session');
    }

    const totalScore = this.calculateTotalScore(session.scores);
    const percentageScore = this.calculatePercentageScore(session.scores);
    const recommendations = this.generateRecommendations(session.scores);
    
    // Generate summary based on percentage score
    let summary;
    if (percentageScore >= 80) {
      summary = 'Strong business model with solid foundation across all areas.';
    } else if (percentageScore >= 60) {
      summary = 'Promising business model with some areas needing attention.';
    } else if (percentageScore >= 40) {
      summary = 'Business model shows potential but requires significant development.';
    } else {
      summary = 'Business model needs substantial work across multiple areas.';
    }

    return {
      totalScore,
      percentageScore,
      areaScores: { ...session.scores },
      recommendations,
      summary,
      completedAt: new Date().toISOString(),
      responses: { ...session.responses }
    };
  }

  getAreaDescription(area) {
    const descriptions = {
      'industry-background': 'Industry Background & Expertise',
      'product-validation': 'Product Validation',
      'market-demand': 'Market Demand',
      'business-goals': 'Business Goals & Strategy',
      'traction-momentum': 'Traction & Momentum',
      'hypothesis-prioritization': 'Hypothesis Prioritization'
    };
    return descriptions[area] || area;
  }

  generateStructuredReport(session) {
    if (!this.isSessionComplete(session)) {
      throw new Error('Cannot generate structured report for incomplete session');
    }

    const report = this.generateReport(session);
    const maxPossible = this.assessmentAreas.length * 2;

    // Identify strengths (areas with score 2)
    const strengths = [];
    Object.entries(session.scores).forEach(([area, score]) => {
      if (score === 2) {
        strengths.push(this.getAreaDescription(area));
      }
    });

    // Identify concerns (areas with score 0)
    const concerns = [];
    Object.entries(session.scores).forEach(([area, score]) => {
      if (score === 0) {
        concerns.push(this.getAreaDescription(area));
      }
    });

    // Generate priority hypotheses from high-priority recommendations
    const priorityHypotheses = [];
    const highPriorityRecs = report.recommendations.filter(rec => rec.priority === 'high');
    highPriorityRecs.slice(0, 3).forEach(rec => {
      const hypothesisMap = {
        'industry-background': 'Validate that domain expertise gaps can be addressed through advisors or partnerships',
        'product-validation': 'Test core product assumptions with target users through prototyping',
        'market-demand': 'Validate market size and customer willingness to pay for the solution',
        'business-goals': 'Confirm revenue model viability and path to profitability',
        'traction-momentum': 'Identify sustainable customer acquisition channels',
        'hypothesis-prioritization': 'Test the most critical business model assumptions first'
      };
      priorityHypotheses.push(hypothesisMap[rec.area] || `Validate key assumptions in ${this.getAreaDescription(rec.area)}`);
    });

    // Generate recommended actions from all recommendations
    const recommendedActions = report.recommendations.map(rec => rec.message);

    // Create markdown report
    let markdownContent = `BUSINESS MODEL VALIDATION REPORT
================================
Overall Health Score: ${report.totalScore}/${maxPossible} points (${report.percentageScore}%)

Strengths:`;

    if (strengths.length > 0) {
      strengths.forEach(strength => {
        markdownContent += `\n- ${strength}`;
      });
    } else {
      markdownContent += `\n- No areas scored at maximum level`;
    }

    markdownContent += `\n\nConcerns:`;
    if (concerns.length > 0) {
      concerns.forEach(concern => {
        markdownContent += `\n- ${concern}`;
      });
    } else {
      markdownContent += `\n- No critical gaps identified`;
    }

    markdownContent += `\n\nUnanswered/Unclear:`;
    // For areas with score 1 (adequate but not clear)
    const unclearAreas = [];
    Object.entries(session.scores).forEach(([area, score]) => {
      if (score === 1) {
        unclearAreas.push(this.getAreaDescription(area));
      }
    });
    
    if (unclearAreas.length > 0) {
      unclearAreas.forEach(area => {
        markdownContent += `\n- ${area} needs additional clarity and validation`;
      });
    } else {
      markdownContent += `\n- All areas have clear evidence or identified gaps`;
    }

    markdownContent += `\n\nPriority Hypotheses to Test (Next 6 Months):`;
    if (priorityHypotheses.length > 0) {
      priorityHypotheses.forEach((hypothesis, index) => {
        markdownContent += `\n${index + 1}. ${hypothesis}`;
      });
    } else {
      markdownContent += `\n1. Continue strengthening existing validated areas`;
      markdownContent += `\n2. Expand market validation and customer research`;
      markdownContent += `\n3. Test scalability of current business model`;
    }

    markdownContent += `\n\nRecommended Actions:`;
    if (recommendedActions.length > 0) {
      recommendedActions.forEach(action => {
        markdownContent += `\n- ${action}`;
      });
    } else {
      markdownContent += `\n- Continue monitoring and optimizing current strong performance`;
    }

    return markdownContent;
  }

  saveStructuredReport(session, projectName) {
    if (!projectName || typeof projectName !== 'string') {
      throw new Error('Project name is required and must be a string');
    }

    const markdownContent = this.generateStructuredReport(session);
    
    // Create projects directory path relative to current working directory
    const projectsDir = path.join(process.cwd(), 'projects');
    const projectDir = path.join(projectsDir, projectName);
    const filePath = path.join(projectDir, 'business-model-validation.md');
    
    // Create directories if they don't exist
    if (!fs.existsSync(projectsDir)) {
      fs.mkdirSync(projectsDir, { recursive: true });
    }
    
    if (!fs.existsSync(projectDir)) {
      fs.mkdirSync(projectDir, { recursive: true });
    }
    
    // Save the report
    fs.writeFileSync(filePath, markdownContent, 'utf8');
    
    return filePath;
  }
}
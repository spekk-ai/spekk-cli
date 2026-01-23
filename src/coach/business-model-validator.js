export class BusinessModelValidator {
  constructor() {
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
}
export interface PrivacySection {
  heading: string;
  paragraphs: string[];
  items?: string[];
}

export interface Dictionary {
  common: {
    brand: string;
    loading: string;
    save: string;
    saving: string;
    cancel: string;
    submit: string;
    submitting: string;
    backHome: string;
    privacyPolicy: string;
    languageLabel: string;
    themeLabel: string;
    themeLight: string;
    themeDark: string;
    showPassword: string;
    hidePassword: string;
    passwordStrengthLabel: string;
    passwordWeak: string;
    passwordFair: string;
    passwordStrong: string;
    passwordMinHint: string;
    passwordMismatch: string;
    emailLabel: string;
    emailPlaceholder: string;
    passwordLabel: string;
    confirmPasswordLabel: string;
    newPasswordLabel: string;
    currentPasswordLabel: string;
  };
  nav: {
    chat: string;
    profile: string;
    logout: string;
    login: string;
    register: string;
  };
  footer: {
    copyright: string;
    privacyLink: string;
  };
  landing: {
    eyebrow: string;
    title: string;
    tagline: string;
    ctaRegister: string;
    ctaLogin: string;
    ctaOpenChat: string;
    ctaContinueOnboarding: string;
    capabilitiesHeading: string;
    capabilities: {
      title: string;
      description: string;
    }[];
    stepsHeading: string;
    steps: {
      step: string;
      title: string;
      description: string;
    }[];
  };
  register: {
    title: string;
    submitButton: string;
    hasAccount: string;
    loginLink: string;
    privacyPrefix: string;
    successTitle: string;
    successMessage: string;
    devTokenLabel: string;
    devVerifyLink: string;
    goToVerify: string;
  };
  verifyEmail: {
    title: string;
    tokenLabel: string;
    tokenPlaceholder: string;
    submitButton: string;
    verifying: string;
    successTitle: string;
    successMessage: string;
    goToLogin: string;
  };
  login: {
    title: string;
    submitButton: string;
    forgotPasswordLink: string;
    noAccount: string;
    registerLink: string;
    sessionExpired: string;
  };
  forgotPassword: {
    title: string;
    submitButton: string;
    successMessage: string;
    devResetLink: string;
    backToLogin: string;
  };
  resetPassword: {
    title: string;
    tokenLabel: string;
    tokenPlaceholder: string;
    submitButton: string;
    successMessage: string;
    goToLogin: string;
  };
  onboarding: {
    title: string;
    heightLabel: string;
    weightLabel: string;
    ageLabel: string;
    genderLabel: string;
    genderMale: string;
    genderFemale: string;
    activityLabel: string;
    activityLight: string;
    activityModerate: string;
    activityHeavy: string;
    submitButton: string;
  };
  profile: {
    title: string;
    summaryHeading: string;
    avatarAlt: string;
    dailyCalories: string;
    protein: string;
    carbs: string;
    fat: string;
    fiber: string;
    kcalUnit: string;
    gramUnit: string;
    editProfileHeading: string;
    macroTargetsHeading: string;
    recalcNote: string;
    saveProfileButton: string;
    profileSaved: string;
    changePasswordHeading: string;
    changePasswordButton: string;
    passwordChanged: string;
    privacyHeading: string;
    privacyLinkText: string;
    deleteAccountHeading: string;
    deleteAccountButton: string;
    deleteDialogTitle: string;
    deleteDialogQuestion: string;
    deleteDialogWarning: string;
    deleteDialogPasswordLabel: string;
    deleteDialogConfirm: string;
    deleteDialogDeleting: string;
  };
  privacy: {
    title: string;
    updatedLabel: string;
    sections: PrivacySection[];
  };
  chat: {
    title: string;
    subtitle: string;
    targetSummary: string;
    inputPlaceholder: string;
    sendButton: string;
    shellNotice: string;
    editProfileLink: string;
  };
  notFound: {
    title: string;
    description: string;
    backHome: string;
  };
}

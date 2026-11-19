import type { Dictionary } from "@/lib/i18n/types";

export const enDictionary: Dictionary = {
  common: {
    brand: "Nouriko",
    loading: "Loading...",
    save: "Save",
    saving: "Saving...",
    cancel: "Cancel",
    submit: "Submit",
    submitting: "Processing...",
    backHome: "Back to Home",
    privacyPolicy: "Privacy Policy",
    languageLabel: "Language",
    themeLabel: "Theme",
    themeLight: "Light",
    themeDark: "Dark",
    showPassword: "Show",
    hidePassword: "Hide",
    passwordStrengthLabel: "Password strength",
    passwordWeak: "Weak",
    passwordFair: "Fair",
    passwordStrong: "Strong",
    passwordMinHint: "At least 8 characters and not too common.",
    passwordMismatch: "Password confirmation does not match.",
    emailLabel: "Email",
    emailPlaceholder: "name@example.com",
    passwordLabel: "Password",
    confirmPasswordLabel: "Confirm Password",
    newPasswordLabel: "New Password",
    currentPasswordLabel: "Current Password",
  },
  nav: {
    chat: "Chat",
    profile: "Profile",
    logout: "Log out",
    login: "Sign in",
    register: "Sign up",
  },
  footer: {
    copyright: "Nouriko — Daily Nutrition Assistant",
    privacyLink: "Privacy Policy",
  },
  landing: {
    eyebrow: "DAILY NUTRITION ASSISTANT",
    title: "Nouriko",
    tagline:
      "Daily meal recommendations and nutrition estimates tailored to your ingredients, budget, and food photos.",
    ctaRegister: "Sign up",
    ctaLogin: "Sign in",
    ctaOpenChat: "Open Chat",
    ctaContinueOnboarding: "Complete Profile",
    capabilitiesHeading: "Core Capabilities",
    capabilities: [
      {
        title: "Ingredient-Based Meals",
        description:
          "Enter the ingredients you already have at home to get practical meal ideas with estimated calories and macros.",
      },
      {
        title: "Budget-Aware Planning",
        description:
          "Set a daily or meal budget to receive balanced meal recommendations that fit your spending limit.",
      },
      {
        title: "Food Photo Analysis",
        description:
          "Upload a photo of your meal to estimate its ingredients, calories, protein, carbohydrates, fat, and fiber.",
      },
    ],
    stepsHeading: "How It Works",
    steps: [
      {
        step: "01",
        title: "Create an Account & Profile",
        description:
          "Sign up with your email, verify your account, and enter your body profile to calculate daily calorie and macro targets.",
      },
      {
        step: "02",
        title: "Share Ingredients, Budget, or a Photo",
        description:
          "Open the chat page and describe what you want to eat in plain language or upload a food photo.",
      },
      {
        step: "03",
        title: "Review & Track Your Meals",
        description:
          "Check meal suggestions against your daily targets and update your profile anytime your needs change.",
      },
    ],
  },
  register: {
    title: "Create Account",
    submitButton: "Sign up",
    hasAccount: "Already have an account?",
    loginLink: "Sign in",
    privacyPrefix: "By signing up, you can review how we handle data in our",
    successTitle: "Check Your Email",
    successMessage:
      "Your account has been created. Please open the verification link sent to your email before signing in.",
    devTokenLabel: "Local verification link:",
    devVerifyLink: "Verify Email Now",
    goToVerify: "Enter Verification Token",
  },
  verifyEmail: {
    title: "Verify Email",
    tokenLabel: "Verification Token",
    tokenPlaceholder: "Paste your verification token here",
    submitButton: "Verify Email",
    verifying: "Verifying email...",
    successTitle: "Email Verified",
    successMessage: "Your email has been verified. You can now sign in.",
    goToLogin: "Continue to Sign in",
  },
  login: {
    title: "Sign in",
    submitButton: "Sign in",
    forgotPasswordLink: "Forgot password?",
    noAccount: "Do not have an account?",
    registerLink: "Sign up",
    sessionExpired: "Your session has ended. Please sign in again.",
  },
  forgotPassword: {
    title: "Forgot Password",
    submitButton: "Send Reset Link",
    successMessage:
      "If that email is registered, we have sent a link to reset your password.",
    devResetLink: "Open Reset Password Page",
    backToLogin: "Back to Sign in",
  },
  resetPassword: {
    title: "Reset Password",
    tokenLabel: "Reset Token",
    tokenPlaceholder: "Paste your reset token here",
    submitButton: "Save New Password",
    successMessage:
      "Your password has been updated. Please sign in with your new password.",
    goToLogin: "Sign in Now",
  },
  onboarding: {
    title: "Profile Details",
    heightLabel: "Height (cm)",
    weightLabel: "Weight (kg)",
    ageLabel: "Age (years)",
    genderLabel: "Gender",
    genderMale: "Male",
    genderFemale: "Female",
    activityLabel: "Activity Level",
    activityLight: "Light",
    activityModerate: "Moderate",
    activityHeavy: "Heavy",
    submitButton: "Save",
  },
  profile: {
    title: "Profile & Account Settings",
    summaryHeading: "Daily Target Summary",
    avatarAlt: "User profile avatar",
    dailyCalories: "Calorie Target",
    protein: "Protein",
    carbs: "Carbs",
    fat: "Fat",
    fiber: "Fiber",
    kcalUnit: "kcal",
    gramUnit: "g",
    editProfileHeading: "Body Metrics",
    macroTargetsHeading: "Daily Calorie & Macro Targets",
    recalcNote:
      "Changing body metrics automatically recalculates daily targets unless you enter custom target values.",
    saveProfileButton: "Save Changes",
    profileSaved: "Profile updated successfully.",
    changePasswordHeading: "Change Password",
    changePasswordButton: "Update Password",
    passwordChanged: "Password updated successfully.",
    privacyHeading: "Privacy & Data",
    privacyLinkText: "Read Nouriko Privacy Policy",
    deleteAccountHeading: "Delete Account",
    deleteAccountButton: "Delete Account",
    deleteDialogTitle: "Delete Account",
    deleteDialogQuestion: "Are you sure you want to delete this account?",
    deleteDialogWarning:
      "This action cannot be undone. All account data and saved history will be permanently deleted.",
    deleteDialogPasswordLabel: "Enter Password",
    deleteDialogConfirm: "Delete",
    deleteDialogDeleting: "Deleting...",
  },
  privacy: {
    title: "Privacy Policy",
    updatedLabel: "Last updated: November 19, 2026",
    sections: [
      {
        heading: "1. Data We Collect",
        paragraphs: [
          "Nouriko only collects the information necessary to provide personalized daily nutrition assistance.",
        ],
        items: [
          "Account data: email address and securely hashed password for authentication.",
          "Body profile data: height, weight, age, gender, activity level, and daily calorie and macro targets.",
          "App preferences: selected language (Indonesian or English) and appearance theme (Light or Dark).",
          "Saved meals that you explicitly choose to save.",
        ],
      },
      {
        heading: "2. How We Use Your Data",
        paragraphs: [
          "Your body profile is used to calculate personalized daily calorie and macronutrient targets. Your email is used for account verification, sign-in, and password recovery. We do not sell or share your personal data with third parties for advertising.",
        ],
      },
      {
        heading: "3. Data Storage and Retention",
        paragraphs: [
          "Account, profile, and saved meal records are stored in an isolated per-account database. Chat messages are ephemeral during your session, and uploaded food photos are processed only for nutrition analysis without permanent storage on our servers.",
        ],
      },
      {
        heading: "4. Your Rights and Account Deletion",
        paragraphs: [
          "You have the right to access, update, and delete your personal data at any time from your Profile page.",
        ],
        items: [
          "Update your body metrics, nutrition targets, language, theme, and password at any time.",
          "Permanently delete your account from the Delete Account section on your Profile page after verifying your password. Deleting your account permanently removes your profile and saved meals.",
        ],
      },
      {
        heading: "5. Not a Medical Tool",
        paragraphs: [
          "Nutrition estimates, calorie calculations, and meal suggestions provided by Nouriko are for general informational purposes only. Nouriko is not a medical device and does not replace advice from a physician, clinical dietitian, or licensed healthcare professional.",
        ],
      },
      {
        heading: "6. Legal Basis",
        paragraphs: [
          "Personal data processing in Nouriko aligns with Indonesia Law No. 27 of 2022 on Personal Data Protection (UU PDP), including user consent, purpose limitation, secure storage, and the right to erasure.",
        ],
      },
    ],
  },
  chat: {
    title: "Nutrition Chat",
    subtitle: "Chat workspace shell ready for the next session.",
    targetSummary: "Your daily target",
    inputPlaceholder:
      "Type your available ingredients, budget, or meal request...",
    sendButton: "Send",
    shellNotice:
      "Your account and profile are ready. Interactive nutrition chat will be enabled in the next phase.",
    editProfileLink: "Edit Profile",
  },
  notFound: {
    title: "Page Not Found",
    description:
      "The page you are looking for is unavailable or the link may have changed.",
    backHome: "Back to Home",
  },
};

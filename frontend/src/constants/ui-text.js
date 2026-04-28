export const appMeta = {
  appName: "卡玛聊天",
  defaultTitle: "卡玛聊天",
};

export const authPageText = {
  eyebrow: "欢迎回来",
  title: "登录后开始聊天",
  subtitle:
    "用手机号登录或注册，和朋友继续聊。",
  highlights: [
    {
      title: "手机号登录",
      description: "支持密码和短信验证码。",
    },
    {
      title: "快速注册",
      description: "填好昵称和手机号就能开始。",
    },
    {
      title: "安心使用",
      description: "下次打开可以继续上次的聊天。",
    },
  ],
  panelTitle: "欢迎进入卡玛聊天",
  panelSubtitle: "选择一种方式登录。",
  tabs: {
    password: "账号登录",
    sms: "短信登录",
    register: "注册账号",
  },
  form: {
    phoneLabel: "手机号",
    phonePlaceholder: "请输入中国大陆手机号",
    passwordLabel: "密码",
    passwordPlaceholder: "请输入登录密码",
    nicknameLabel: "昵称",
    nicknamePlaceholder: "请输入昵称",
    smsCodeLabel: "短信验证码",
    smsCodePlaceholder: "请输入验证码",
  },
  buttons: {
    login: "登录",
    loginBySms: "验证码登录",
    register: "注册并进入",
    sendCode: "发送验证码",
  },
  tipsTitle: "可用方式",
  tipsList: [
    "账号密码登录",
    "短信验证码登录",
    "手机号注册",
    "记住登录",
  ],
  footerNote: "你可以用密码、短信验证码登录，也可以直接注册。",
};

export const authMessages = {
  invalidPhone: "请输入合法的中国大陆手机号",
  passwordRequired: "请输入密码",
  nicknameRequired: "请输入昵称",
  smsCodeRequired: "请输入短信验证码",
  formNotReady: "表单尚未准备完成，请稍后重试",
  loginSuccess: "登录成功",
  loginFailed: "登录失败，请检查手机号和密码",
  smsLoginFailed: "短信登录失败，请核对验证码后重试",
  registerSuccess: "注册成功",
  registerFailed: "注册失败，请稍后重试",
  smsCodeSent: "验证码已发送，请留意短信",
  smsCodeFailed: "验证码发送失败，请稍后重试",
  cooldownSuffix: " 秒后重试",
};

export const homePageText = {
  eyebrow: "首页",
  title: "欢迎进入卡玛聊天",
  subtitle:
    "去看消息、好友和群聊，也可以完善你的个人信息。",
  viewProfileAction: "查看个人信息",
  logoutAction: "退出登录",
  summaryItems: {
    identity: "身份",
    status: "账号情况",
    joinedAt: "注册时间",
    contact: "手机号",
  },
  profileTitle: "个人信息",
  profileSubtitle: "这里是你的昵称、手机号和其他个人信息。",
  profileFields: {
    uuid: "账号",
    nickname: "昵称",
    telephone: "手机号",
    email: "邮箱",
    gender: "性别",
    birthday: "生日",
    createdAt: "注册时间",
    role: "身份",
    status: "账号情况",
  },
  signatureTitle: "个性签名",
  signatureFallback: "这个账号还没有设置个性签名。",
  capabilityTitle: "去看看",
  capabilityList: [
    "好友",
    "消息",
    "聊天",
    "群聊",
  ],
  identity: {
    admin: "管理员",
    user: "普通用户",
  },
  status: {
    normal: "正常",
    disabled: "已禁用",
  },
  gender: {
    male: "男",
    female: "女",
  },
  emptyValue: "未填写",
  logoutSuccess: "已退出登录",
};

export const routeTitles = {
  auth: "登录注册",
  home: "首页",
  messages: "我的消息",
  profile: "个人信息",
  friends: "我的好友",
  groups: "我的群聊",
  createGroup: "创建群聊",
};

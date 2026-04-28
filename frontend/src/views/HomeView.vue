<script setup>
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";

import { homePageText } from "../constants/ui-text";
import { clearStoredUser, getStoredUser } from "../utils/storage";

const router = useRouter();
const recentSectionRef = ref(null);

const currentUser = computed(() => getStoredUser() || {});

const uiText = {
  projectName: "KamaChat",
  navbarEyebrow: "聊天首页",
  heroEyebrow: "欢迎使用",
  welcomeTitle: "欢迎回来，开始你的聊天之旅",
  welcomeSubtitle: "看看消息、好友和群聊。",
  featureTitle: "常用功能",
  featureSubtitle: "常用的聊天功能都在这里。",
  recentTitle: "最近聊天",
  recentSubtitle: "继续上次没聊完的话题。",
};

const displayName = computed(() => currentUser.value.nickname || "Kama 用户");
const identityText = computed(() => {
  return currentUser.value.is_admin === 1 ? homePageText.identity.admin : homePageText.identity.user;
});
const statusText = computed(() => {
  return currentUser.value.status === 0 ? homePageText.status.normal : homePageText.status.disabled;
});

const summaryItems = computed(() => {
  return [
    {
      label: "身份",
      value: identityText.value,
    },
    {
      label: "账号情况",
      value: statusText.value,
    },
    {
      label: "手机号",
      value: currentUser.value.telephone || "未填写",
    },
  ];
});

const recentSessions = [
  {
    id: "session-1",
    name: "产品讨论组",
    message: "今晚把首页再调整一下，明早一起看。",
    time: "09:40",
    unread: 3,
    tone: "peach",
  },
  {
    id: "session-2",
    name: "林小满",
    message: "我已经把新的好友备注同步上去了，你看一下。",
    time: "昨天",
    unread: 1,
    tone: "cream",
  },
  {
    id: "session-3",
    name: "群提醒",
    message: "保持在线，就能及时收到新消息。",
    time: "周一",
    unread: 0,
    tone: "rose",
  },
];

function findRouteByName(name) {
  return router.getRoutes().find((route) => route.name === name);
}

const featureRouteMap = computed(() => ({
  messages: findRouteByName("messages"),
  friends: findRouteByName("friends"),
  groups: findRouteByName("groups"),
  createGroup: findRouteByName("createGroup"),
  profile: findRouteByName("profile"),
}));

const featureCards = computed(() => {
  return [
    {
      key: "messages",
      glyph: "聊",
      title: "我的消息",
      description: "查看最近聊天与未读消息",
      actionType: featureRouteMap.value.messages ? "route" : "section",
      actionTarget: featureRouteMap.value.messages || recentSectionRef,
    },
    {
      key: "friends",
      glyph: "友",
      title: "我的好友",
      description: "查看好友和新的申请",
      actionType: featureRouteMap.value.friends ? "route" : "pending",
      actionTarget: featureRouteMap.value.friends,
    },
    {
      key: "groups",
      glyph: "群",
      title: "我的群聊",
      description: "查看已加入的群聊",
      actionType: featureRouteMap.value.groups ? "route" : "pending",
      actionTarget: featureRouteMap.value.groups,
    },
    {
      key: "createGroup",
      glyph: "建",
      title: "创建群聊",
      description: "邀请好友开启新的群聊",
      actionType: featureRouteMap.value.createGroup ? "route" : "pending",
      actionTarget: featureRouteMap.value.createGroup,
    },
    {
      key: "profile",
      glyph: "我",
      title: "个人信息",
      description: "修改头像和昵称",
      actionType: featureRouteMap.value.profile ? "route" : "pending",
      actionTarget: featureRouteMap.value.profile,
    },
  ];
});

function getInitials(name) {
  return String(name || "K").trim().slice(0, 1).toUpperCase();
}

function scrollToElement(targetRef) {
  targetRef?.value?.scrollIntoView({
    behavior: "smooth",
    block: "start",
  });
}

function handleFeatureClick(feature) {
  if (feature.actionType === "route" && feature.actionTarget) {
    router.push({ name: feature.actionTarget.name });
    return;
  }

  if (feature.actionType === "section") {
    scrollToElement(feature.actionTarget);
    return;
  }

  ElMessage.info(`${feature.title}暂时还不能打开`);
}

function logout() {
  clearStoredUser();
  ElMessage.success(homePageText.logoutSuccess);
  router.push("/auth");
}
</script>

<template>
  <main class="page-shell home-page">
    <header class="glass-card home-navbar">
      <div class="home-navbar__brand">
        <span class="home-navbar__logo">K</span>
        <div>
          <p class="home-navbar__eyebrow">{{ uiText.navbarEyebrow }}</p>
          <h1 class="home-navbar__title">{{ uiText.projectName }}</h1>
        </div>
      </div>

      <div class="home-navbar__actions">
        <div class="home-navbar__user">
          <el-avatar :size="42" :src="currentUser.avatar || ''" class="home-navbar__avatar">
            {{ getInitials(displayName) }}
          </el-avatar>
          <div>
            <p class="home-navbar__name">{{ displayName }}</p>
            <p class="home-navbar__meta">{{ identityText }}</p>
          </div>
        </div>

        <el-button type="primary" plain @click="logout">
          {{ homePageText.logoutAction }}
        </el-button>
      </div>
    </header>

    <section class="glass-card home-board">
      <section class="home-board__section home-hero">
        <div class="home-hero__content">
          <p class="eyebrow">{{ uiText.heroEyebrow }}</p>
          <h2 class="display-title home-hero__title">{{ uiText.welcomeTitle }}</h2>
          <p class="muted-copy home-hero__copy">{{ uiText.welcomeSubtitle }}</p>

          <div class="home-hero__summary">
            <article v-for="item in summaryItems" :key="item.label" class="hero-stat">
              <p class="hero-stat__label">{{ item.label }}</p>
              <p class="hero-stat__value">{{ item.value }}</p>
            </article>
          </div>
        </div>

        <div class="home-hero__visual" aria-hidden="true">
          <div class="hero-visual__phone">
            <div class="hero-visual__bar"></div>
            <div class="hero-visual__bubble hero-visual__bubble--primary"></div>
            <div class="hero-visual__bubble hero-visual__bubble--secondary"></div>
            <div class="hero-visual__bubble hero-visual__bubble--accent"></div>
          </div>
          <div class="hero-visual__float hero-visual__float--left">在线中</div>
          <div class="hero-visual__float hero-visual__float--right">新消息</div>
        </div>
      </section>

      <section class="home-board__section">
        <div class="board-heading">
          <div>
            <p class="eyebrow">{{ uiText.featureTitle }}</p>
            <h2 class="section-title">快捷操作</h2>
          </div>
          <p class="board-heading__copy">{{ uiText.featureSubtitle }}</p>
        </div>

        <div class="feature-grid">
          <button
            v-for="feature in featureCards"
            :key="feature.key"
            type="button"
            class="feature-card"
            @click="handleFeatureClick(feature)"
          >
            <span class="feature-card__icon">{{ feature.glyph }}</span>
            <h3 class="feature-card__title">{{ feature.title }}</h3>
            <p class="feature-card__copy">{{ feature.description }}</p>
            <span class="feature-card__arrow">→</span>
          </button>
        </div>
      </section>

      <section ref="recentSectionRef" class="home-board__section recent-panel">
        <header class="recent-panel__header">
          <div>
            <p class="eyebrow">{{ uiText.recentTitle }}</p>
            <h2 class="section-title">最近聊天</h2>
          </div>
          <p class="recent-panel__hint">{{ uiText.recentSubtitle }}</p>
        </header>

        <div class="recent-list">
          <article v-for="session in recentSessions" :key="session.id" class="recent-item">
            <div class="recent-item__avatar" :class="`is-${session.tone}`">
              {{ getInitials(session.name) }}
            </div>

            <div class="recent-item__body">
              <div class="recent-item__row">
                <h3 class="recent-item__name">{{ session.name }}</h3>
                <span class="recent-item__time">{{ session.time }}</span>
              </div>
              <p class="recent-item__message">{{ session.message }}</p>
            </div>

            <span v-if="session.unread > 0" class="recent-item__badge">{{ session.unread }}</span>
          </article>
        </div>
      </section>
    </section>
  </main>
</template>

<style scoped>
.home-page {
  display: grid;
  gap: 20px;
  padding-top: 8px;
}

.home-navbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 16px 20px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(18px);
}

.home-navbar__brand,
.home-navbar__actions,
.home-navbar__user {
  display: flex;
  align-items: center;
}

.home-navbar__brand,
.home-navbar__user {
  gap: 14px;
}

.home-navbar__actions {
  gap: 16px;
}

.home-navbar__logo {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  border-radius: 14px;
  background: linear-gradient(135deg, #f7bf96 0%, #ec9b66 100%);
  color: #ffffff;
  font-size: 18px;
  font-weight: 800;
  box-shadow: 0 10px 18px rgba(188, 90, 52, 0.18);
}

.home-navbar__eyebrow,
.home-navbar__meta {
  margin: 0;
  font-size: 12px;
  color: var(--kc-muted);
}

.home-navbar__title,
.home-navbar__name {
  margin: 0;
  font-size: 18px;
  font-weight: 800;
}

.home-board {
  display: grid;
  gap: 0;
  padding: 28px;
  background:
    radial-gradient(circle at 14% 12%, rgba(255, 255, 255, 0.78) 0%, rgba(255, 255, 255, 0) 32%),
    linear-gradient(180deg, rgba(255, 249, 244, 0.96) 0%, rgba(255, 255, 255, 0.94) 100%);
}

.home-board__section + .home-board__section {
  margin-top: 28px;
  padding-top: 28px;
  border-top: 1px solid rgba(232, 224, 214, 0.9);
}

.home-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(260px, 360px);
  gap: 24px;
  align-items: center;
}

.home-hero__title {
  max-width: 620px;
}

.home-hero__copy {
  max-width: 560px;
  margin-top: 16px;
}

.home-hero__summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-top: 24px;
}

.hero-stat {
  padding: 16px;
  border: 1px solid rgba(232, 224, 214, 0.9);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.78);
}

.hero-stat__label {
  margin: 0;
  color: var(--kc-muted);
  font-size: 12px;
  font-weight: 600;
}

.hero-stat__value {
  margin: 8px 0 0;
  font-size: 15px;
  font-weight: 800;
}

.home-hero__visual {
  position: relative;
  min-height: 250px;
}

.hero-visual__phone {
  position: absolute;
  right: 22px;
  bottom: 0;
  width: 208px;
  height: 236px;
  padding: 20px 18px;
  border: 1px solid rgba(231, 214, 199, 0.9);
  border-radius: 32px;
  background: rgba(255, 255, 255, 0.85);
  box-shadow: 0 24px 46px rgba(145, 87, 58, 0.08);
}

.hero-visual__bar {
  width: 64px;
  height: 10px;
  margin: 0 auto 18px;
  border-radius: 999px;
  background: rgba(227, 154, 105, 0.22);
}

.hero-visual__bubble {
  border-radius: 18px;
  background: rgba(244, 235, 228, 0.96);
}

.hero-visual__bubble + .hero-visual__bubble {
  margin-top: 12px;
}

.hero-visual__bubble--primary {
  width: 72%;
  height: 44px;
  background: linear-gradient(135deg, #f7c7a3 0%, #f1a874 100%);
}

.hero-visual__bubble--secondary {
  width: 86%;
  height: 56px;
  margin-left: auto;
}

.hero-visual__bubble--accent {
  width: 64%;
  height: 42px;
}

.hero-visual__float {
  position: absolute;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 92px;
  padding: 10px 14px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.92);
  color: #a55331;
  font-size: 13px;
  font-weight: 700;
  box-shadow: 0 14px 28px rgba(145, 87, 58, 0.08);
}

.hero-visual__float--left {
  top: 18px;
  left: 12px;
}

.hero-visual__float--right {
  right: 0;
  bottom: 30px;
}

.board-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
}

.board-heading__copy {
  margin: 0;
  max-width: 280px;
  color: var(--kc-muted);
  line-height: 1.7;
  text-align: right;
}

.feature-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 16px;
}

.feature-card {
  position: relative;
  display: grid;
  gap: 12px;
  min-height: 186px;
  padding: 20px;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 22px;
  background: rgba(255, 255, 255, 0.82);
  text-align: left;
  cursor: pointer;
  transition:
    transform 0.22s ease,
    box-shadow 0.22s ease,
    border-color 0.22s ease;
}

.feature-card:hover {
  transform: translateY(-4px);
  border-color: rgba(214, 153, 119, 0.48);
  box-shadow: 0 20px 34px rgba(145, 87, 58, 0.1);
}

.feature-card__icon {
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  border-radius: 14px;
  background: linear-gradient(135deg, #fff0e2 0%, #f9c9a2 100%);
  color: #a65431;
  font-size: 18px;
  font-weight: 800;
}

.feature-card__title {
  margin: 0;
  font-size: 18px;
  font-weight: 800;
  letter-spacing: 0;
}

.feature-card__copy {
  margin: 0;
  color: var(--kc-muted);
  line-height: 1.68;
}

.feature-card__arrow {
  margin-top: auto;
  color: var(--kc-accent);
  font-size: 18px;
  font-weight: 700;
}

.recent-panel__header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
}

.recent-panel__hint {
  margin: 0;
  max-width: 240px;
  color: var(--kc-muted);
  font-size: 12px;
  line-height: 1.6;
  text-align: right;
}

.recent-list {
  display: grid;
  gap: 12px;
  margin-top: 18px;
}

.recent-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 14px;
  align-items: center;
  padding: 16px;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 18px;
  background: rgba(248, 245, 241, 0.7);
}

.recent-item__avatar {
  display: grid;
  place-items: center;
  width: 46px;
  height: 46px;
  border-radius: 16px;
  font-size: 16px;
  font-weight: 800;
  color: #8c4a2d;
}

.recent-item__avatar.is-peach {
  background: #ffd9c0;
}

.recent-item__avatar.is-cream {
  background: #ffeccc;
}

.recent-item__avatar.is-rose {
  background: #ffd7d9;
}

.recent-item__body {
  min-width: 0;
}

.recent-item__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.recent-item__name {
  margin: 0;
  font-size: 15px;
  font-weight: 800;
}

.recent-item__time {
  color: var(--kc-muted);
  font-size: 12px;
}

.recent-item__message {
  margin: 8px 0 0;
  color: var(--kc-muted);
  line-height: 1.6;
}

.recent-item__badge {
  min-width: 24px;
  padding: 4px 8px;
  border-radius: 999px;
  background: var(--kc-accent);
  color: #ffffff;
  font-size: 12px;
  font-weight: 700;
  text-align: center;
}

@media (max-width: 1100px) {
  .feature-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 980px) {
  .home-navbar,
  .home-hero,
  .recent-panel__header,
  .board-heading {
    flex-direction: column;
    align-items: flex-start;
  }

  .home-navbar__actions,
  .home-hero__summary,
  .feature-grid {
    width: 100%;
  }

  .home-navbar__actions {
    justify-content: space-between;
  }

  .home-hero {
    grid-template-columns: 1fr;
  }

  .home-hero__summary {
    grid-template-columns: 1fr;
  }

  .feature-grid {
    grid-template-columns: 1fr;
  }

  .board-heading__copy,
  .recent-panel__hint {
    max-width: none;
    text-align: left;
  }
}

@media (max-width: 720px) {
  .home-navbar__actions,
  .home-navbar__user {
    width: 100%;
  }

  .home-navbar__actions {
    flex-direction: column;
    align-items: stretch;
  }

  .recent-item {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .recent-item__badge {
    justify-self: start;
    margin-left: 60px;
  }
}

@media (max-width: 640px) {
  .home-navbar,
  .home-board {
    padding: 18px;
  }

  .hero-visual__phone {
    position: relative;
    right: auto;
    margin: 0 auto;
  }

  .hero-visual__float--left {
    left: 0;
  }

  .hero-visual__float--right {
    right: 0;
  }
}
</style>

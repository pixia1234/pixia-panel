import { Button } from "@heroui/button";
import { Input } from "@heroui/input";
import { Card, CardBody, CardHeader } from "@heroui/card";
import { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import toast from 'react-hot-toast';
import { isWebViewFunc } from '@/utils/panel';
import { siteConfig, getCachedConfig } from '@/config/site';
import { title } from "@/components/primitives";
import DefaultLayout from "@/layouts/default";
import { login, LoginData, checkCaptcha } from "@/api";


interface LoginForm {
  username: string;
  password: string;
  captchaId: string;
  turnstileToken?: string;
}



export default function IndexPage() {
  const [form, setForm] = useState<LoginForm>({
    username: "",
    password: "",
    captchaId: "",
    turnstileToken: "",
  });
  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState<Partial<LoginForm>>({});
  const [showTurnstile, setShowTurnstile] = useState(false);
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState("");
  const [pendingLogin, setPendingLogin] = useState(false);
  const navigate = useNavigate();
  const turnstileContainerRef = useRef<HTMLDivElement>(null);
  const turnstileWidgetRef = useRef<any>(null);
  const [isWebView, setIsWebView] = useState(false);
  // 检测是否在WebView中运行
  useEffect(() => {
    setIsWebView(isWebViewFunc());
  }, []);

  // 加载 Cloudflare Turnstile 配置
  useEffect(() => {
    const loadTurnstileConfig = async () => {
      try {
        const enabled = await getCachedConfig('turnstile_enabled');
        const captchaType = await getCachedConfig('captcha_type');
        const siteKey = await getCachedConfig('turnstile_site_key');
        const useTurnstile = (captchaType || '').toUpperCase() === 'TURNSTILE' || enabled === 'true';
        setTurnstileEnabled(useTurnstile);
        setTurnstileSiteKey(siteKey || "");
      } catch (error) {
        setTurnstileEnabled(false);
        setTurnstileSiteKey("");
      }
    };
    loadTurnstileConfig();
  }, []);

  useEffect(() => {
    if (showTurnstile) {
      renderTurnstile();
    }
  }, [showTurnstile, turnstileEnabled, turnstileSiteKey]);

  const loadTurnstileScript = (): Promise<void> => {
    return new Promise((resolve, reject) => {
      if ((window as any).turnstile) {
        resolve();
        return;
      }
      const existing = document.getElementById('turnstile-script');
      if (existing) {
        existing.addEventListener('load', () => resolve());
        return;
      }
      const script = document.createElement('script');
      script.id = 'turnstile-script';
      script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';
      script.async = true;
      script.defer = true;
      script.onload = () => resolve();
      script.onerror = () => reject(new Error('Turnstile 脚本加载失败'));
      document.body.appendChild(script);
    });
  };

  const renderTurnstile = async () => {
    if (!turnstileEnabled || !turnstileSiteKey || !turnstileContainerRef.current) return;
    await loadTurnstileScript();
    const turnstile = (window as any).turnstile;
    if (!turnstile) return;

    if (turnstileWidgetRef.current) {
      turnstile.reset(turnstileWidgetRef.current);
      return;
    }

    const isDarkMode = document.documentElement.classList.contains('dark') || 
                       document.documentElement.getAttribute('data-theme') === 'dark' ||
                       window.matchMedia('(prefers-color-scheme: dark)').matches;

    turnstileWidgetRef.current = turnstile.render(turnstileContainerRef.current, {
      sitekey: turnstileSiteKey,
      theme: isDarkMode ? 'dark' : 'light',
      callback: (token: string) => {
        setForm(prev => ({ ...prev, turnstileToken: token }));
        if (pendingLogin) {
          setPendingLogin(false);
          setLoading(true);
          performLogin();
        }
      },
      'error-callback': () => {
        setForm(prev => ({ ...prev, turnstileToken: "" }));
        toast.error('验证码加载失败，请重试');
      },
      'expired-callback': () => {
        setForm(prev => ({ ...prev, turnstileToken: "" }));
      }
    });
  };
  // 验证表单
  const validateForm = (): boolean => {
    const newErrors: Partial<LoginForm> = {};

    if (!form.username.trim()) {
      newErrors.username = '请输入用户名';
    }

    if (!form.password.trim()) {
      newErrors.password = '请输入密码';
    } else if (form.password.length < 6) {
      newErrors.password = '密码长度至少6位';
    }


    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  // 处理输入变化
  const handleInputChange = (field: keyof LoginForm, value: string) => {
    setForm(prev => ({ ...prev, [field]: value }));
    // 清除该字段的错误
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: undefined }));
    }
  };

  // 执行登录请求
  const performLogin = async () => {


    try {
      const loginData: LoginData = {
        username: form.username.trim(),
        password: form.password,
        captchaId: form.captchaId,
        turnstileToken: form.turnstileToken,
      };

      const response = await login(loginData);
      
      if (response.code !== 0) {
        toast.error(response.msg || "登录失败");
        if (turnstileWidgetRef.current && (window as any).turnstile) {
          (window as any).turnstile.reset(turnstileWidgetRef.current);
        }
        setForm(prev => ({ ...prev, turnstileToken: "" }));
        return;
      }

      // 检查是否需要强制修改密码
      if (response.data.requirePasswordChange) {
        localStorage.setItem('token', response.data.token);
        localStorage.setItem("role_id", response.data.role_id.toString());
        localStorage.setItem("name", response.data.name);
        localStorage.setItem("admin", (response.data.role_id === 0).toString());
        toast.success('检测到默认密码，即将跳转到修改密码页面');
        navigate("/change-password");
        return;
      }

      // 保存登录信息
      localStorage.setItem('token', response.data.token);
      localStorage.setItem("role_id", response.data.role_id.toString());
      localStorage.setItem("name", response.data.name);
      localStorage.setItem("admin", (response.data.role_id === 0).toString());

      // 登录成功
      toast.success('登录成功');
      navigate("/dashboard");

    } catch (error) {
      console.error('登录错误:', error);
      toast.error("网络错误，请稍后重试");
    } finally {
      setLoading(false);
    }
  };

  const handleLogin = async () => {
    if (!validateForm()) return;

    setLoading(true);

    try {
      // 先检查是否需要验证码
      const checkResponse = await checkCaptcha();
      
      if (checkResponse.code !== 0) {
        toast.error("检查验证码状态失败，请重试" + checkResponse.msg);
        setLoading(false);
        return;
      }

      const required = checkResponse.data !== 0;

      if (!required) {
        setShowTurnstile(false);
        setPendingLogin(false);
        await performLogin();
        return;
      }

      if (!turnstileEnabled || !turnstileSiteKey) {
        toast.error('未配置 Cloudflare 验证码');
        setLoading(false);
        return;
      }

      setShowTurnstile(true);
      await renderTurnstile();
      if (!form.turnstileToken) {
        setPendingLogin(true);
        setLoading(false);
        return;
      }
      await performLogin();
    } catch (error) {
      console.error('检查验证码状态错误:', error);
      toast.error("网络错误，请稍后重试" + error);
      setLoading(false);
    }
  };


  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !loading) {
      handleLogin();
    }
  };

  return (
    <DefaultLayout>
      <section className="flex flex-col items-center justify-center gap-4 py-4 sm:py-8 md:py-10 pb-20 min-h-[calc(100dvh-120px)] sm:min-h-[calc(100dvh-200px)]">
        <div className="w-full max-w-md px-4 sm:px-0">
          <Card className="w-full">
            <CardHeader className="pb-0 pt-6 px-6 flex-col items-center">
              <h1 className={title({ size: "sm" })}>登陆</h1>
              <p className="text-small text-default-500 mt-2">请输入您的账号信息</p>
            </CardHeader>
            <CardBody className="px-6 py-6">
              <div className="flex flex-col gap-4">
                <Input
                  label="用户名"
                  placeholder="请输入用户名"
                  value={form.username}
                  onChange={(e) => handleInputChange('username', e.target.value)}
                  onKeyDown={handleKeyPress}
                  variant="bordered"
                  isDisabled={loading}
                  isInvalid={!!errors.username}
                  errorMessage={errors.username}
                />
                
                <Input
                  label="密码"
                  placeholder="请输入密码"
                  type="password"
                  value={form.password}
                  onChange={(e) => handleInputChange('password', e.target.value)}
                  onKeyDown={handleKeyPress}
                  variant="bordered"
                  isDisabled={loading}
                  isInvalid={!!errors.password}
                />

                {showTurnstile && turnstileEnabled && (
                  <div className="flex justify-center">
                    <div ref={turnstileContainerRef} />
                  </div>
                )}

                
                <Button
                  color="primary"
                  size="lg"
                  onClick={handleLogin}
                  isLoading={loading}
                  disabled={loading}
                  className="mt-2"
                >
                  {loading ? (showTurnstile ? "验证中..." : "登录中...") : "登录"}
                </Button>
              </div>
            </CardBody>
          </Card>
        </div>


      {/* 版权信息 - 固定在底部，不占据布局空间 */}
      
               <div className="fixed inset-x-0 bottom-4 text-center py-4">
               <p className="text-xs text-gray-400 dark:text-gray-500">
                 Powered by{' '}
                 <a 
                   href="https://github.com/pixia1234/pixia-panel" 
                   target="_blank" 
                   rel="noopener noreferrer"
                   className="text-gray-500 dark:text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
                 >
                   pixia-panel
                 </a>
               </p>
               <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
                 v{ isWebView ? siteConfig.app_version : siteConfig.version}
               </p>
             </div>
      
   

      </section>
    </DefaultLayout>
  );
}

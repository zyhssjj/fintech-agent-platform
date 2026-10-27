import React, { useState } from 'react';
import './CurvedInput.css';

// 定义所有可用的 Props
export interface CurvedInputProps {
  /** 占位符文本 */
  placeholder?: string;
  /** 按钮上的文字，如果不提供则不显示按钮 */
  buttonText?: string;
  /** 预设主题：'dark' | 'light' | 自定义对象 */
  theme?: 'dark' | 'light' | { 
    backgroundColor: string; 
    textColor: string; 
    borderColor: string; 
    buttonColor: string; 
    buttonTextColor: string; 
  };
  /** 弯曲程度（角度），单位 deg，默认 28 */
  bend?: number;
  /** 容器高度（px），默认 64 */
  height?: number;
  /** 容器宽度（px），默认 450 */
  width?: number;
  /** 输入框类型，如 'text', 'email', 'search' */
  type?: string;
  /** 圆角半径（px），默认 18 */
  cornerRadius?: number;
  /** 边框宽度（px），默认 1.5 */
  borderWidth?: number;
  /** 字体大小（px），默认 16 */
  fontSize?: number;
  /** 背景颜色（覆盖 theme） */
  backgroundColor?: string;
  /** 文字颜色（覆盖 theme） */
  textColor?: string;
  /** 边框颜色（覆盖 theme） */
  borderColor?: string;
  /** 按钮背景色（覆盖 theme） */
  buttonColor?: string;
  /** 按钮文字颜色（覆盖 theme） */
  buttonTextColor?: string;
  /** 阴影大小：'none' | 'sm' | 'md' | 'lg'，默认 'none' */
  shadowSize?: 'none' | 'sm' | 'md' | 'lg';
  /** 是否显示按钮（默认为 false） */
  showButton?: boolean;
  /** 是否显示图标（暂时只作为占位） */
  showIcon?: boolean;
  /** 提交回调 */
  onSubmit?: (value: string) => void;
}

const CurvedInput: React.FC<CurvedInputProps> = ({
  placeholder = 'Enter text...',
  buttonText = '',
  theme = 'light',
  bend = 28,
  height = 64,
  width = 450,
  type = 'text',
  cornerRadius = 18,
  borderWidth = 1.5,
  fontSize = 16,
  backgroundColor,
  textColor,
  borderColor,
  buttonColor,
  buttonTextColor,
  shadowSize = 'none',
  showButton = false,
  showIcon = false,
  onSubmit,
}) => {
  const [value, setValue] = useState('');

  // 解析 theme 或使用自定义颜色
  const getThemeColors = () => {
    if (typeof theme === 'object') {
      return {
        bg: theme.backgroundColor || '#ffffff',
        text: theme.textColor || '#333333',
        border: theme.borderColor || '#cccccc',
        btnBg: theme.buttonColor || '#007bff',
        btnText: theme.buttonTextColor || '#ffffff',
      };
    }
    // 预设主题
    const dark = {
      bg: '#1e1e2e',
      text: '#cdd6f4',
      border: '#313244',
      btnBg: '#a6e3a1',
      btnText: '#11111b',
    };
    const light = {
      bg: '#ffffff',
      text: '#1e1e2e',
      border: '#d9e0ee',
      btnBg: '#1e1e2e',
      btnText: '#ffffff',
    };
    return theme === 'dark' ? dark : light;
  };

  const themeColors = getThemeColors();

  // 合并自定义颜色（如果提供）
  const finalBg = backgroundColor || themeColors.bg;
  const finalText = textColor || themeColors.text;
  const finalBorder = borderColor || themeColors.border;
  const finalBtnBg = buttonColor || themeColors.btnBg;
  const finalBtnText = buttonTextColor || themeColors.btnText;

  // 阴影映射
  const shadowMap = {
    none: 'none',
    sm: '0 2px 4px rgba(0,0,0,0.1)',
    md: '0 4px 8px rgba(0,0,0,0.2)',
    lg: '0 8px 16px rgba(0,0,0,0.3)',
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (onSubmit) onSubmit(value);
  };

  // 计算弯曲偏移（使用 skewY 实现视觉弯曲效果）
  const bendTransform = `skewY(${bend > 0 ? -bend : 0}deg)`;

  return (
    <form
      className="curved-input-form"
      style={{
        height: `${height}px`,
        width: `${width}px`,
        borderRadius: `${cornerRadius}px`,
        border: `${borderWidth}px solid ${finalBorder}`,
        backgroundColor: finalBg,
        boxShadow: shadowMap[shadowSize] || 'none',
        transform: bendTransform,
        transition: 'transform 0.2s ease',
      }}
      onSubmit={handleSubmit}
    >
      {showIcon && (
        <span className="curved-input-icon" style={{ color: finalText }}>
          {/* 这里可以放一个搜索图标，暂时用文字代替 */}
          🔍
        </span>
      )}
      <input
        type={type}
        className="curved-input-field"
        placeholder={placeholder}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        style={{
          fontSize: `${fontSize}px`,
          color: finalText,
          backgroundColor: 'transparent',
          border: 'none',
          outline: 'none',
          flex: 1,
          padding: '0 16px',
          height: '100%',
          minWidth: 0,
        }}
      />
      {showButton && buttonText && (
        <button
          type="submit"
          className="curved-input-button"
          style={{
            backgroundColor: finalBtnBg,
            color: finalBtnText,
            fontSize: `${fontSize * 0.9}px`,
            padding: '0 20px',
            border: 'none',
            borderRadius: `${cornerRadius}px`,
            cursor: 'pointer',
            height: '100%',
            whiteSpace: 'nowrap',
            transition: 'background-color 0.2s',
          }}
          onMouseEnter={(e) => (e.currentTarget.style.opacity = '0.85')}
          onMouseLeave={(e) => (e.currentTarget.style.opacity = '1')}
        >
          {buttonText}
        </button>
      )}
    </form>
  );
};

export default CurvedInput;
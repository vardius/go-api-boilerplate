import React from "react";
import {defineMessages, useIntl} from "react-intl";
import {Link as ReachLink, useRouteMatch} from "react-router-dom";
import {
  Avatar,
  Box,
  Flex,
  Heading,
  HStack,
  Icon,
  Input,
  InputGroup,
  InputLeftElement,
  Link,
  Menu,
  MenuButton,
  MenuDivider,
  MenuItem,
  MenuList,
  Text,
  useColorModeValue,
} from "@chakra-ui/core";
import {FaExternalLinkAlt} from "react-icons/fa";
import {brandColors} from "src/theme/theme";
import {useUser} from "src/hooks";
import getPath from "src/routes";
import LoginDrawerButton from "src/components/common/LoginDrawerButton";

const messages = defineMessages({
  title: {
    id: "app.header.title",
    defaultMessage: "go-api-boilerplate",
  },
  search: {
    id: "app.header.search",
    defaultMessage: "Search...",
  },
  menu: {
    id: "app.header.menu",
    defaultMessage: "Menu",
  },
  home: {
    id: "app.header.home",
    defaultMessage: "Home",
  },
  auth_tokens: {
    id: "app.header.auth_tokens",
    defaultMessage: "Auth Tokens",
  },
  clients: {
    id: "app.header.clients",
    defaultMessage: "Client Credentials",
  },
  mail: {
    id: "app.header.mail",
    defaultMessage: "MailBox",
  },
  mysql: {
    id: "app.header.mysql",
    defaultMessage: "MYSQL",
  },
  logout: {
    id: "app.header.logout",
    defaultMessage: "Logout",
  },
});

export interface MenuLinkProps {
  to: string;
  exact?: boolean;
  isExternal?: boolean;
  children?: React.ReactNode;
}

const MenuLink = ({children, to, exact, isExternal}: MenuLinkProps) => {
  const match = useRouteMatch({
    path: to,
    exact,
  });

  if (isExternal) {
    return (
      <Link href={to} mt={{base: 4, md: 0}} mr={6} display="block" isExternal>
        <HStack>
          <Text>{children}</Text>
          <Icon as={FaExternalLinkAlt} ml={1}/>
        </HStack>
      </Link>
    );
  }

  if (match) {
    return (
      <Text mt={{base: 4, md: 0}} mr={6} display="block">
        {children}
      </Text>
    );
  }

  return (
    <Link as={ReachLink} to={to} mt={{base: 4, md: 0}} mr={6} display="block">
      {children}
    </Link>
  );
};

const Header = () => {
  const intl = useIntl();
  const [isVisible, setVisible] = React.useState(false);
  const [user, setUser] = useUser();

  const color = useColorModeValue(brandColors.light, brandColors.dark);

  const handleToggle = () => setVisible(!isVisible);

  return (
    <Flex
      as="nav"
      align="center"
      justify="space-between"
      wrap="wrap"
      padding="1.0rem"
      borderBottomWidth="1px"
      borderColor={color.primary}
    >
      <Flex align="center" mr={5}>
        <Heading as="h1" size="lg" letterSpacing={"-.1rem"}>
          {intl.formatMessage(messages.title)}
        </Heading>
      </Flex>

      <Box display={{base: "block", md: "none"}} onClick={handleToggle}>
        <svg
          fill={color.primary}
          width="12px"
          viewBox="0 0 20 20"
          xmlns="http://www.w3.org/2000/svg"
        >
          <title>{intl.formatMessage(messages.menu)}</title>
          <path d="M0 3h20v2H0V3zm0 6h20v2H0V9zm0 6h20v2H0v-2z"/>
        </svg>
      </Box>

      <Box
        display={{base: isVisible ? "block" : "none", md: "flex"}}
        width={{base: "full", md: "auto"}}
        alignItems="center"
        flexGrow={1}
      >
        <MenuLink to={getPath("home")} exact>
          {intl.formatMessage(messages.home)}
        </MenuLink>
        <MenuLink to={`https://maildev.${window.location.hostname}`} isExternal>
          {intl.formatMessage(messages.mail)}
        </MenuLink>
      </Box>

      <Box
        display={{base: isVisible ? "block" : "none", md: "block"}}
        mt={{base: 4, md: 0}}
      >
        <HStack spacing={2} align="center">
          <InputGroup flexGrow={1}>
            <InputLeftElement zIndex={0} children={<Icon name="search"/>}/>
            <Input flex="1" placeholder={intl.formatMessage(messages.search)}/>
          </InputGroup>
          {user ? (
            <Menu>
              <MenuButton>
                <Avatar name={user.email} src="#"/>
              </MenuButton>
              <MenuList>
                <MenuLink to={getPath("auth_tokens")} exact>
                  <MenuItem>
                    {intl.formatMessage(messages.auth_tokens)}
                  </MenuItem>
                </MenuLink>
                <MenuItem>
                  <MenuLink to={getPath("client_credentials")} exact>
                    {intl.formatMessage(messages.clients)}
                  </MenuLink>
                </MenuItem>
                <MenuDivider/>
                <MenuItem
                  onClick={() => {
                    setUser(null);
                  }}
                >
                  {intl.formatMessage(messages.logout)}
                </MenuItem>
              </MenuList>
            </Menu>
          ) : (
            <LoginDrawerButton/>
          )}
        </HStack>
      </Box>
    </Flex>
  );
};

export default Header;

import React from "react";
import { defineMessages, useIntl } from "react-intl";
import { Link as ReachLink, useRouteMatch } from "react-router-dom";
import {
  Box,
  Stack,
  Heading,
  Flex,
  InputGroup,
  InputLeftElement,
  Icon,
  Input,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  Link,
  Text,
  HStack,
  useColorModeValue,
} from "@chakra-ui/core";
import { FaExternalLinkAlt } from "react-icons/fa";
import { brandColors } from "src/theme/theme";
import { useUser } from "src/hooks";
import { LoginDrawerButton } from "src/components/login";

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

const MenuLink = ({ children, to, exact, isExternal }: MenuLinkProps) => {
  const match = useRouteMatch({
    path: to,
    exact,
  });

  if (isExternal) {
    return (
      <Link href={to} isExternal>
        {children}
        <Icon as={FaExternalLinkAlt} mx="4px" boxSize="12px" />
      </Link>
    );
  }

  if (match) {
    return <Text>{children}</Text>;
  }

  return (
    <Link as={ReachLink} to="/">
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
      padding="1.5rem"
      borderBottomWidth="1px"
      borderColor={color.primary}
    >
      <Flex align="center" mr={5}>
        <Heading as="h1" size="lg" letterSpacing={"-.1rem"}>
          {intl.formatMessage(messages.title)}
        </Heading>
      </Flex>

      <Box display={{ base: "block", md: "none" }} onClick={handleToggle}>
        <svg
          width="12px"
          viewBox="0 0 20 20"
          xmlns="http://www.w3.org/2000/svg"
        >
          <title>{intl.formatMessage(messages.menu)}</title>
          <path d="M0 3h20v2H0V3zm0 6h20v2H0V9zm0 6h20v2H0v-2z" />
        </svg>
        {intl.formatMessage(messages.mail)}
        {intl.formatMessage(messages.mysql)}
        {intl.formatMessage(messages.mail)}
        {intl.formatMessage(messages.mysql)}
      </Box>

      <HStack
        spacing={4}
        display={{ sm: isVisible ? "block" : "none", md: "flex" }}
        width={{ sm: "full", md: "auto" }}
        alignItems="center"
        flexGrow={1}
      >
        <MenuLink to="/" exact>
          {intl.formatMessage(messages.home)}
        </MenuLink>
        <MenuLink to="https://maildev.go-api-boilerplate.local/" isExternal>
          {intl.formatMessage(messages.mail)}
        </MenuLink>
        <MenuLink to="https://phpmyadmin.go-api-boilerplate.local/" isExternal>
          {intl.formatMessage(messages.mysql)}
        </MenuLink>
      </HStack>

      <Box
        display={{ sm: isVisible ? "block" : "none", md: "block" }}
        mt={{ base: 4, md: 0 }}
      >
        <Stack spacing={2} align="center" isInline>
          <InputGroup flexGrow={1}>
            <InputLeftElement zIndex={0} children={<Icon name="search" />} />
            <Input flex="1" placeholder={intl.formatMessage(messages.search)} />
          </InputGroup>
          {user ? (
            <Menu>
              <MenuButton as={Text}>{user.email}</MenuButton>
              <MenuList>
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
            <LoginDrawerButton />
          )}
        </Stack>
      </Box>
    </Flex>
  );
};

export default Header;
